package webclient

import (
	"fmt"
	"github.com/yudeguang/distributedhttpproxy/common"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

//间隔上报类
//var reporter = &clientReporter{}
//工作任务类
var worker = &clsClientWorker{}

//外部传递的clientname,这个可能各个Agent有重复的，真正的在服务器端是还要加上本机IP和路由出口ip
var userClientName = ""

//客户端的id
var globalOnlyId string = ""

//代理的地址
var globalProxyAddr string = ""

//服务端tcp服务地址
var globalServerAddr string = ""

//程序启动时间喝进程id
var globalStartTime string
var globalProcessId string

//日志对象
var pLogger *common.BasicLogger = nil

//开始工作主函数
func Client_start(clientName, clientAddr, serverAddr string) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	disableFastEditMode()
	//初始化一些参数保留起来
	userClientName = clientName
	globalProxyAddr = clientAddr
	globalServerAddr = serverAddr
	globalStartTime = common.GetNowTime()
	globalProcessId = fmt.Sprint(os.Getpid())
	pLogger = common.NewLogger(common.GetExeBaseName() + ".log")
	pLogger.Log("**********CLIENT程序启动**************")
	pLogger.Log("程序路径:", filepath.Join(common.GetExePath(), common.GetExeName()))
	pLogger.Log("当前时间:", common.GetNowTime())
	pLogger.Log("进程ID:", globalProcessId)
	pLogger.Log("客户端标识:" + globalOnlyId)
	pLogger.Log("服务器地址:", globalServerAddr)
	//开始工作,连接
	worker.Start()
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
