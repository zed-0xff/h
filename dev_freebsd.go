package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const DIOCGMEDIASIZE = 0x40086481 // from <sys/disk.h>

func getDeviceSize(devicePath string) (int64, error) {
	file, err := os.Open(devicePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open device: %v", err)
	}
	defer file.Close()

	var size int64
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(DIOCGMEDIASIZE),
		uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return 0, fmt.Errorf("ioctl failed: %v", errno)
	}

	return size, nil
}
