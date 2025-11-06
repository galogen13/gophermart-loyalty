//go:build windows

package classification

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func IsRetriableSyscallError(reqErr syscall.Errno) bool {

	switch reqErr {
	case windows.WSAECONNREFUSED,
		windows.WSAECONNRESET,
		windows.WSAETIMEDOUT:
		return true
	}

	return false
}
