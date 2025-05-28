package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func getDeviceAlign(fname string) int {
	return 0 // TODO: check
}

const (
	DKIOCGETBLOCKSIZE  = 0x40046418
	DKIOCGETBLOCKCOUNT = 0x40086419
)

func getDeviceSize(devicePath string) (int64, error) {
	f, err := os.Open(devicePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open device: %v", err)
	}
	defer f.Close()

	var blockSize int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), DKIOCGETBLOCKSIZE, uintptr(unsafe.Pointer(&blockSize)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl DKIOCGETBLOCKSIZE failed: %v", errno)
	}

	var blockCount int64
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), DKIOCGETBLOCKCOUNT, uintptr(unsafe.Pointer(&blockCount)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl DKIOCGETBLOCKCOUNT failed: %v", errno)
	}

	return int64(blockSize) * blockCount, nil
}
