package webservices

import (
	"github.com/yudeguang/distributedhttpproxy/agentcomm"
	"net"
)

type connectMain struct {
	onlyid     string
	conn       net.Conn
	remoteAddr string
}

func (this *connectMain) Init(conn net.Conn, onlyId string, remoteAddr string) {
	this.conn = conn
	this.onlyid = onlyId
	this.remoteAddr = remoteAddr
}
func (this *connectMain) Disconnect() {
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
}

//发送请求连接的命令
func (this *connectMain) SendConnectRequest(sequence int64, proxyAddr string) error {
	var err = agentcomm.WritePackage(this.conn, agentcomm.CMD_REQUEST_REVERSE_CONNECT, proxyAddr, sequence)
	return err
}
