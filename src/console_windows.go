//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleProcessList   = kernel32.NewProc("GetConsoleProcessList")
	procSetConsoleOutputCP      = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP            = kernel32.NewProc("SetConsoleCP")
	msvcrt                      = syscall.NewLazyDLL("msvcrt.dll")
	procGetch                   = msvcrt.NewProc("_getch")
)

func initConsole() {
	const utf8CodePage = 65001
	procSetConsoleOutputCP.Call(uintptr(utf8CodePage))
	procSetConsoleCP.Call(uintptr(utf8CodePage))
}

func shouldPauseOnExit() bool {
	processIDs := make([]uint32, 4)
	result, _, _ := procGetConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&processIDs[0])),
		uintptr(len(processIDs)),
	)
	return result > 0 && result <= 2
}

func waitForExit() {
	fmt.Fprintln(os.Stdout, "按任意键关闭...")
	if _, _, err := procGetch.Call(); err == syscall.Errno(0) {
		return
	}
	_, _ = bufio.NewReader(os.Stdin).ReadByte()
}
