package webservices

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/yudeguang/distributedhttpproxy/agentcomm"
	"github.com/yudeguang/distributedhttpproxy/common"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//服务的版本号,也做服务的标记，上报的Client中必须要包含这个才有效
var mServerFlag string = "test"

//服务端使用的tcp端口,http端口是在tcp端口上加1
var nTcpListenPort = 8888

//日志对象
var pLogger *common.BasicLogger = nil

//启动函数
func Server_start(client_version string, portNum int) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	disableFastEditMode()
	mServerFlag = client_version
	nTcpListenPort = portNum
	pLogger = common.NewLogger(common.GetExeBaseName() + ".log")
	pLogger.Log("**********SERVER程序启动**************")
	pLogger.Log("程序路径:", filepath.Join(common.GetExePath(), common.GetExeName()))
	pLogger.Log("当前时间:", common.GetNowTime())
	pLogger.Log("进程ID:", os.Getpid())
	pLogger.Log("服务器标记:", mServerFlag)
	//创建sqlite数据库文件
	var err = pDBHelper.DBOpen()
	if err != nil {
		pLogger.LogExit("连接数据库失败:", err)
	}
	pLogger.Log("启动服务.......")
	pProxyManager.Init()
	go runBeegoServer()
	time.Sleep(500 * time.Millisecond)
	runTCPServer()
}

//启动Beego服务
func runBeegoServer() {
	pLogger.Log("启动WEB服务,端口:", nTcpListenPort+1)
	beego.BConfig.Listen.HTTPPort = nTcpListenPort + 1
	beego.BConfig.Listen.HTTPAddr = "127.0.0.1"
	beego.BConfig.AppName = "17vinsoft"
	beego.BConfig.RunMode = "dev"
	beego.BConfig.CopyRequestBody = true
	registBeegoFuncMap()
	var viewPath = "./views"
	beego.SetStaticPath("/", viewPath)
	beego.SetStaticPath("/js/", filepath.Join(viewPath, "js"))
	beego.SetStaticPath("/css/", filepath.Join(viewPath, "css"))
	//注册beego路由
	beego.AutoRouter(&AgentController{})
	//beego.Router("/request/*",&ProxyController{},"*:Proxy")
	//注册beego函数
	beego.Run()
	pLogger.LogExit("WEB服务运行结束...")
}

//启动TCP服务
func runTCPServer() {
	pLogger.Log("启动TCP服务,端口:", nTcpListenPort)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", nTcpListenPort))
	if err != nil {
		pLogger.LogExit("启动TCP服务Listen失败:", err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Microsecond)
				continue
			}
			pLogger.LogExit("TCP服务Accept失败:", err)
		}
		go processTCPConnect(conn)
	}
}

//处理tcp链接
func processTCPConnect(conn net.Conn) {
	var hostPort = conn.RemoteAddr().String()
	var logPrint = func(info string) {
		pLogger.Log("[" + hostPort + "]" + info)
	}
	logPrint("收到TCP连接...")
	head, text, err := agentcomm.ReadPackage(conn)
	if err != nil {
		logPrint("读取数据失败:" + err.Error())
		conn.Close()
		return
	}
	logPrint(fmt.Sprintf("读取到数据,CmdID:0x%X,ReverseId:%d,Data:%s", head.CmdId, head.ReverseId, text))
	//连接的第一个请求只可能是CMD_CONNECT_MAIN或CMD_REQUEST_REVERSE_CONNECT
	if head.CmdId == agentcomm.CMD_CONNECT_MAIN {
		//处理完数据之后再循环接收数据
		var onlyId = ""
		remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		switchData, err := processMainConnectData(text, remoteIP)
		if err != nil {
			logPrint(err.Error())
			goto MAIN_CONNECT_END
		}
		switchData.RemoteAddr = conn.RemoteAddr().String()
		err = agentcomm.WritePackage(conn, agentcomm.CMD_SUCCESS, "")
		if err != nil {
			logPrint("回复主连数据错误:" + err.Error())
			goto MAIN_CONNECT_END
		}
		onlyId = switchData.OnlyId
		//回复成功,并加入到缓存连接
		pConnectCache.Add(conn, switchData)
		//循环读取数据,这里就不要回复了，回复数据就是请求代理连接功能
		for {
			head, text, err = agentcomm.ReadPackage(conn, 10)
			if err != nil {
				logPrint("接收主连接数据错误:" + err.Error())
				goto MAIN_CONNECT_END
			}
			if head.CmdId != agentcomm.CMD_CONNECT_MAIN {
				logPrint(fmt.Sprintf("主连接上收到命令:0x%X 无效,断开连接", head.CmdId))
				goto MAIN_CONNECT_END
			}
			processMainConnectData(text, remoteIP)
		}
	MAIN_CONNECT_END:
		pConnectCache.Delete(onlyId)
		pDBHelper.SetIsActiveByOnlyId(onlyId, 0)
		conn.Close()
		return
	} else if head.CmdId == agentcomm.CMD_REPLY_REVERSE_CONNECT {
		//反向连接来了
		ok := pProxyManager.AddReverseReponse(head.ReverseId, conn)
		if !ok {
			conn.Close()
		}
		return
	} else {
		logPrint(fmt.Sprintf("连接/首包数据命令类型:0x%.8X不支持,断开链接", head.CmdId))
	}
	conn.Close()
	return
}

//处理上报的数据
func processMainConnectData(text string, remoteIP string) (*agentcomm.TagSwitchData, error) {
	switchData, err := agentcomm.SwitchDataFromJson(text)
	if err != nil {
		return nil, fmt.Errorf("请求JSON转换为SwitchData错误:%v", err)
	}
	if switchData.OnlyId == "" || !strings.Contains(switchData.OnlyId, mServerFlag) {
		return switchData, fmt.Errorf("OnlyId(%s)与版本(%s)不匹配,断开链接", switchData.OnlyId, mServerFlag)
	}
	//onlyid加上remoteip这样才能唯一
	switchData.OnlyId += "@" + remoteIP
	//更新信息到数据库
	pDBHelper.UpdateAgentRecord(switchData)
	return switchData, nil
}

//禁用快速编辑模式
var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode = modkernel32.NewProc("SetConsoleMode")
)

func disableFastEditMode() {
	hStdin, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		log.Println(err)
		return
	}
	var mode uint32
	err = syscall.GetConsoleMode(hStdin, &mode)
	if err != nil {
		log.Println(err)
		return
	}
	mode = mode & (^uint32(0x0010)) //ENABLE_MOUSE_INPUT
	mode = mode & (^uint32(0x0020)) //ENABLE_INSERT_MODE
	mode = mode & (^uint32(0x0040)) //ENABLE_QUICK_EDIT_MODE
	procSetConsoleMode.Call(uintptr(hStdin), uintptr(mode))
}
