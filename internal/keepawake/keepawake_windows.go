package keepawake

import (
	"syscall"
)

const (
	ES_CONTINUOUS       uint32 = 0x80000000
	ES_DISPLAY_REQUIRED uint32 = 0x00000002
	ES_SYSTEM_REQUIRED  uint32 = 0x00000001
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
)

func SetKeepAwake(keepAwake bool) error {
	var state uint32
	if keepAwake {
		state = ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED
	} else {
		state = ES_CONTINUOUS
	}

	ret, _, err := procSetThreadExecutionState.Call(uintptr(state))
	if ret == 0 {
		return err
	}
	return nil
}
