package webservices

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
)

//当前正确使用的客户端
var curInUseClientNames sync.Map

//通过http代码并用GET方式获取数据
//URL表示请求的某个客户端的地址，比如 http://127.0.0.1:8080/index.htm
//groups表示客户端分组信息
func ProxyGet(URL string, groups ...string) (HTML string, ClientName string, err error) {
	//当前使用的onlyid和proxyaddr存储到一个map里,退出的时候释放
	var usedOnlyId string
	var usedConn net.Conn
	defer func() {
		//为了安全起见,在这个地方就试图释放是否使用的状态
		log.Println("释放代理:", usedOnlyId)
		pProxyManager.FreeOnlyId(usedOnlyId)
	}()
	//定义连接函数
	var dialfunc = func(network, addr string) (net.Conn, error) {
		pLogger.Log("proxyDialFunction连接地址:", network, addr)
		//对这个地址的请求我们不直接连接，用活跃的Client给我们代理处理一下
		onlyId, conn, err := pProxyManager.GetPorxyTCPConnect(addr, groups)
		usedOnlyId = onlyId
		ClientName = onlyId
		usedConn = conn
		if onlyId == "" {
			pLogger.Log("没有找到合适的onlyId代理客户端:" + addr)
		} else {
			if err == nil {
				pLogger.Log(network + ":查找到onlyId=" + onlyId + " addr:" + addr + " Conn连接成功...")
			} else {
				pLogger.Log(network + ":查找到onlyId=" + onlyId + " addr:" + addr + " 失败:" + err.Error())
			}
		}
		return conn, err
	}
	var proxyHttpClient = &http.Client{
		Transport: &http.Transport{
			Dial:            dialfunc,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := proxyHttpClient.Get(URL) //代理方法，自动从多个客户端中找一个来访问
	if err != nil {
		HTML = err.Error()
		return
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		HTML = err.Error()
	} else {
		HTML = string(data)
	}
	//立即关闭socket不再使用了,每次要都新连接
	proxyHttpClient.CloseIdleConnections()
	return
}
