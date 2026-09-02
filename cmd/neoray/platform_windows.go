//go:build windows

package main

import "syscall"

var (
	winmmDLL            = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod = winmmDLL.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmmDLL.NewProc("timeEndPeriod")
)

// https://learn.microsoft.com/en-us/windows/win32/api/timeapi/nf-timeapi-timebeginperiod
func BeginHighresTimer() {
	procTimeBeginPeriod.Call(uintptr(1))
}

// https://learn.microsoft.com/en-us/windows/win32/api/timeapi/nf-timeapi-timeendperiod
func EndHighresTimer() {
	procTimeEndPeriod.Call(uintptr(1))
}
