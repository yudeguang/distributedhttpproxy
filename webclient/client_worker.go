package webclient

import (
	"fmt"
	"github.com/yudeguang/distributedhttpproxy/agentcomm"
	"net"
	"sync"
	"time"
)

type clsClientWorker struct {
	conn     net.Conn
	stopFlag bool
}

func (this *clsClientWorker) Start() {
	for {
		this.doLoopWork()
		time.Sleep(2 * time.Second)
	}
}

//暂停 该函数必须在Start后才有效
func (this *clsClientWorker) Pause() {
	this.stopFlag = true
}

//继续工作，该函数必须在Start后才有效
func (this *clsClientWorker) Restart() {
	this.stopFlag = false
}
func (this *clsClientWorker) doLoopWork() {
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
	pLogger.Log("开始连接到主服务器....")
	var err = this.connectServer()
	if err != nil {
		pLogger.Log("TCP连接到服务器失败:", err)
		return
	}
	pLogger.Log("TCP已连接,开始循环读取请求...")
	//这里用一个线程读取一个发送和一个线程接收,发送线程就是心跳,接收线程就是处理反向连接
	var wg sync.WaitGroup
	wg.Add(2)
	go this.sendHeartThread(this.conn, &wg)
	go this.recvTaskThread(this.conn, &wg)
	wg.Wait()
}

//心跳线程
func (this *clsClientWorker) sendHeartThread(conn net.Conn, pwg *sync.WaitGroup) {
	defer pwg.Done()
	defer conn.Close()
	for {
		if this.stopFlag == false {
			var err = agentcomm.WritePackage(this.conn, agentcomm.CMD_CONNECT_MAIN, createSwitchData().ToString())
			if err != nil {
				pLogger.Log("发送心跳失败:", err)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
}

//接收任务线程
func (this *clsClientWorker) recvTaskThread(conn net.Conn, pwg *sync.WaitGroup) {
	defer pwg.Done()
	defer conn.Close()
	//接收任务线程
	for {
		header, text, err := agentcomm.ReadPackage(conn, 60)
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() { //超时错误,继续
				pLogger.Log("循环读取请求超时,继续读取...")
				continue
			}
			pLogger.Log("循环读取请求失败:", err)
			return
		}
		if header.CmdId == agentcomm.CMD_REQUEST_REVERSE_CONNECT {
			pLogger.Log("收到反向连接请求,连接号:", header.ReverseId, ",连接地址:"+text)
			session := &clsClientSession{}
			go session.Run(header.ReverseId, text)
		} else {
			pLogger.Log(fmt.Sprintf("读取到未处理请求,CMD=0x%X,Data:%s", header.CmdId, text))
		}
	}
}

//连接到服务器
func (this *clsClientWorker) connectServer() error {
	conn, err := net.DialTimeout("tcp", globalServerAddr, time.Second*15)
	if err != nil {
		return fmt.Errorf("连接失败:%v", err)
	}
	err = conn.SetDeadline(time.Now().Add(time.Second * 10))
	if err != nil {
		conn.Close()
		return fmt.Errorf("设置超时失败:%v", err)
	}
	host, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	globalOnlyId = userClientName + "@" + globalProxyAddr + "@" + host
	pLogger.Log("加入本机IP后globalOnlyId=" + globalOnlyId)
	conn.(*net.TCPConn).SetLinger(20)
	//发送我是主链接
	switchData := createSwitchData()
	err = agentcomm.WritePackage(conn,
		agentcomm.CMD_CONNECT_MAIN,
		switchData.ToString())
	if err != nil {
		conn.Close()
		return fmt.Errorf("发送主链接请求失败:%v", err)
	}
	//等待服务端的回馈连接成功
	head, _, err := agentcomm.ReadPackage(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("读取主链接反馈失败:%v", err)
	}
	if head.CmdId != agentcomm.CMD_SUCCESS {
		conn.Close()
		return fmt.Errorf("主链接返回错误命令:%v", head.CmdId)
	}
	//到这里就完全握手成功了
	this.conn = conn
	return nil
}
