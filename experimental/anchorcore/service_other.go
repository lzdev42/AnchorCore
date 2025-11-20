//go:build !windows

package anchorcore

import "syscall"

func dup(fd int) (nfd int, err error) {
	return syscall.Dup(fd)
}
