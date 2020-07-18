package webservices

import (
	"database/sql"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var pProxyManager = &clsProxyManager{}
var plGlobalProxySequence = new(int64)
//当前正在代理的缓存类
type clsProxyManager struct{
	dbQueryFreeLocker sync.Mutex;
	mapProxy map[string]string;
	mapLocker sync.Mutex;
	//以下是反向连接的存储
	mapReverseConn map[int64]chan net.Conn
	mapReverseLocker sync.Mutex;
}
func (this* clsProxyManager) Init(){
	this.mapProxy = make(map[string]string)
	this.mapReverseConn = make(map[int64]chan net.Conn)
}
//根据需要的代理地址,尝试获取一个空闲的代理，并且设置为忙标记
func (this* clsProxyManager) GetAndLockOnlyId(proxyAddr string,lstExculde []string,lstGroups []string) (string,error){
	sqlText := fmt.Sprintf("SELECT OnlyId FROM AgentList WHERE ifnull(IsActive,0)=1 AND ifnull(IsBusy,0)=0 AND ifnull(Disabled,0)=0 AND ProxyAddr='%s'",proxyAddr);
	//处理过滤掉的onlyid
	for i:=0;i<len(lstExculde);i++{
		lstExculde[i] = "'"+lstExculde[i]+"'";
	}
	if len(lstExculde)>0{
		sqlText += " AND OnlyId Not IN ("+strings.Join(lstExculde,",")+")"
	}
	//处理groupname匹配
	if len(lstGroups) == 0{ //没有传入的时候，匹配为空的
		sqlText += " AND ifnull(GroupName,'')=''"
	}else{
		if len(lstGroups)==1 && strings.ToLower(lstGroups[0]) == "all"{
			//使用全部可用,那么久不加条件
		}else{
			//拼接查询
			for i:=0;i<len(lstGroups);i++{
				lstGroups[i] = "'"+lstGroups[i]+"'";
			}
			if len(lstGroups)>0{
				sqlText += " AND GroupName IN ("+strings.Join(lstGroups,",")+")"
			}
		}
	}
	sqlText += " ORDER BY LastUseTime ASC LIMIT 1"
	this.dbQueryFreeLocker.Lock()
	defer this.dbQueryFreeLocker.Unlock();
	var onlyId string
	err := pDBHelper.QueryRow(sqlText).Scan(&onlyId)
	if err != nil{
		if err == sql.ErrNoRows{
			return "",fmt.Errorf("no_valid_proxy_client")
		}
		return "",err;
	}
	//找到了，将这个onlyid设置忙标记并更新调度时间
	nowTime := time.Now().Format("2006-01-02 15:04:05.000")
	pDBHelper.Exec("UPDATE AgentList SET IsBusy=1,LastUseTime=? WHERE OnlyId=?",nowTime,onlyId)
	return onlyId,nil;
}
//释放一个忙标记
func (this* clsProxyManager) FreeOnlyId(onlyId string){
	this.dbQueryFreeLocker.Lock()
	defer this.dbQueryFreeLocker.Unlock();
	if(onlyId != ""){
		pDBHelper.Exec("UPDATE AgentList SET IsBusy=0 WHERE OnlyId=?",onlyId)
	}
}
//根据需要的代理地址，找到一个合适的Agent请求一个反向链接上来
func (this* clsProxyManager) GetPorxyTCPConnect(proxyAddr string,lstGroups []string) (onlyId string, conn net.Conn, err error){
	var lstTryedOnlyId = []string{}
	for i:=0;i<2;i++{
		onlyId = ""
		conn =  nil;
		err = nil;
		onlyId,err = this.GetAndLockOnlyId(proxyAddr,lstTryedOnlyId,lstGroups)
		if err != nil{
			return;
		}
		//这里确定了onlyid,给他发一个反向链接的请求
		conn,err = this.tryCreateRevserveConnect(onlyId,proxyAddr);
		if err != nil{//失败的话释放掉这个连接代理
			this.FreeOnlyId(onlyId)
			lstTryedOnlyId = append(lstTryedOnlyId,onlyId)
			onlyId = "";
			if strings.HasPrefix(err.Error(),"retry:"){
				pLogger.Log("不可用OnlyId,触发尝试下一个,已尝试:",lstTryedOnlyId)
				continue;
			}
		}
		//这里就成功了或是不能重试的
		break;
	}
	return;
}
//发送一个代理请求,看看是否能回应
func (this* clsProxyManager) tryCreateRevserveConnect(onlyId string,proxyAddr string)(net.Conn,error){
	connMain := pConnectCache.Get(onlyId);
	if connMain == nil{
		return nil,fmt.Errorf("retry:没有在ConnectCache中发现OnlyId:%v",onlyId);
	}
	//发送反向连接请求
	revserveId := atomic.AddInt64(plGlobalProxySequence,1);
	//先准备一个管道放在缓存里再发送数据
	cChannel := make(chan net.Conn);
	this.AddReverseRequest(revserveId,cChannel);
	defer func(){
		this.DelReverseRequest(revserveId);
	}();
	//发送反向连接请求
	if err := connMain.SendConnectRequest(revserveId,proxyAddr);err != nil{
		return nil,err;
	}
	//等待接收，并且删除请求
	var newConn net.Conn = nil;
	select {
	case newConn =<-cChannel:
	case <-time.After(time.Second * 8):
	}
	if newConn == nil{
		return nil,fmt.Errorf("等待反向连接超时");
	}
	return newConn,nil;
}
func (this* clsProxyManager) AddReverseRequest(seq int64,ch chan net.Conn){
	this.mapReverseLocker.Lock()
	defer this.mapReverseLocker.Unlock();
	this.mapReverseConn[seq] = ch;
}
func (this* clsProxyManager) DelReverseRequest(seq int64){
	this.mapReverseLocker.Lock()
	defer this.mapReverseLocker.Unlock();
	c,ok := this.mapReverseConn[seq];
	if ok{
		delete(this.mapReverseConn,seq);
		go func(ch chan net.Conn){
			if ch!=nil{
				select{
				case conn := <-ch:
					if(conn !=nil){
						conn.Close()
					}
				default:
					//donothing
				}
			}
		}(c);
	}

}
func (this* clsProxyManager) AddReverseReponse(seq int64,conn net.Conn) bool{
	this.mapReverseLocker.Lock()
	defer this.mapReverseLocker.Unlock();
	ch,ok := this.mapReverseConn[seq];
	if !ok{
		return false;
	}
	select{
		case ch<-conn:
			ok = true;
		default:
			ok = false;
			break;
	}
	return ok;
}