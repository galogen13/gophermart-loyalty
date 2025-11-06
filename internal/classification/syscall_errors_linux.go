//go:build linux

package classification

import "syscall"

func IsRetriableSyscallError(reqErr syscall.Errno) bool {

	switch reqErr {
	case syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.ETIMEDOUT,
		syscall.EAGAIN:
		return true
	}
	return false
}
