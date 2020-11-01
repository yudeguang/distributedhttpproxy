package webservices

import (
	"log"
	"syscall"
)

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
