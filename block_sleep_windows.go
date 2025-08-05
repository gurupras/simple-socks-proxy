//go:build windows

package simplesocksproxy

import (
	"golang.org/x/sys/windows"
)

var (
	kernel32                    = windows.NewLazySystemDLL("kernel32.dll")
	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
)

const (
	ES_AWAYMODE_REQUIRED = 0x00000040 // For media apps, not needed here
	ES_CONTINUOUS        = 0x80000000
	ES_SYSTEM_REQUIRED   = 0x00000001
)

func PreventSleep() error {
	r, _, err := procSetThreadExecutionState.Call(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)
	if r == 0 {
		return err
	}
	return nil
}

func AllowSleep() error {
	r, _, err := procSetThreadExecutionState.Call(ES_CONTINUOUS)
	if r == 0 {
		return err
	}
	return nil
}
