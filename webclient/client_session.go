package webclient

import (
	"fmt"
	"github.com/yudeguang/distributedhttpproxy/agentcomm"
	"io"
	"net"
	"sync"
	"time"
)

type clsClientSession struct {
	reverseId int64  //代理序号
	proxyAddr string //代理的地址
}

func (this *clsClientSession) logInfo(args ...interface{}) {
	info := fmt.Sprint(args...)
	pLogger.Log(fmt.Sprintf("[%d->%s]", this.reverseId, this.proxyAddr), info)
}

//注意这里是主连接下来的,不能关闭this.conn
func (this *clsClientSession) Run(reverseId int64, proxyAddr string) {
	this.reverseId = reverseId
	this.proxyAddr = proxyAddr
	this.logInfo("收到请求代理...")
	//先链接本地目标端口
	localConn, err := net.DialTimeout("tcp", proxyAddr, 3*time.Second)
	if err != nil {
		this.logInfo("连接代理目标失败:" + err.Error())
		return
	}
	defer localConn.Close()
	this.logInfo("连接代理目标成功...")
	//可以向服务端反向链接回去了
	serverConn, err := net.DialTimeout("tcp", globalServerAddr, 3*time.Second)
	if err != nil {
		this.logInfo("反向链接到服务器失败:" + err.Error())
		return
	}
	defer serverConn.Close()
	this.logInfo("反向连接到服务器成功...")
	//发送我回复的ID
	err = agentcomm.WritePackage(serverConn, agentcomm.CMD_REPLY_REVERSE_CONNECT, "", reverseId)
	if err != nil {
		this.logInfo("向服务器回复失败:" + err.Error())
		return
	}
	//两个接口都连接成功,好了,可以交互数据了
	//交换两个socket
	pwg := &sync.WaitGroup{}
	pwg.Add(2)
	go this.swapSocket(localConn, serverConn, pwg)
	go this.swapSocket(serverConn, localConn, pwg)
	pwg.Wait()
	this.logInfo("<<<代理连接结束>>>")
}
func (this *clsClientSession) swapSocket(dst net.Conn, src net.Conn, pwg *sync.WaitGroup) {
	defer func() {
		dst.Close()
		src.Close()
		pwg.Done()
	}()
	_, err := io.Copy(dst, src)
	if err != nil {
		this.logInfo(err.Error())
	}
}
