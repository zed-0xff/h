package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	DIOCGSECTORSIZE   = 0x40046480
	DIOCGMEDIASIZE    = 0x40086481
	DIOCGSTRIPESIZE   = 0x40086483
	DIOCGSTRIPEOFFSET = 0x40086484
)

func getDeviceAlign(fname string) int {
	file, err := os.Open(fname)
	if err != nil {
		panic(fmt.Sprintf("failed to open \"%s\": %v", fname, err))
	}
	defer file.Close()

	var tmp64 int64
	var tmp32 uint32

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(DIOCGSECTORSIZE), uintptr(unsafe.Pointer(&tmp32)))
	if errno == 0 {
		return int(tmp32)
	}
	fmt.Printf("[?] ioctl DIOCGSECTORSIZE failed: %v\n", errno)

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(DIOCGSTRIPESIZE), uintptr(unsafe.Pointer(&tmp64)))
	if errno == 0 {
		return int(tmp64)
	}
	fmt.Printf("[?] ioctl DIOCGSTRIPESIZE failed: %v\n", errno)

	return 0
}

func getDeviceSize(fname string) (int64, error) {
	file, err := os.Open(fname)
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
		return 0, fmt.Errorf("ioctl DIOCGMEDIASIZE failed: %v", errno)
	}

	return size, nil
}
