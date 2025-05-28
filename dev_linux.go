package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	BLKGETSIZE64 = 0x80081272 // ioctl command for getting the size of a block device (in bytes)
)

func getDeviceSize(devicePath string) (int64, error) {
	// Open the device file
	fd, err := syscall.Open(devicePath, syscall.O_RDONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open device: %v", err)
	}
	defer syscall.Close(fd)

	var size int64
	// Perform the ioctl call to get the size of the device
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(BLKGETSIZE64), uintptr(unsafe.Pointer(&size)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl failed: %v", errno)
	}

	return size, nil
}
