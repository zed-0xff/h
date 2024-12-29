//go:build !windows

package main

import (
	"errors"

	"golang.org/x/sys/unix"
)

func getDriveSize(drive string) (int64, error) {
	return 0, errors.New("Not implemented")
}

func findHole(fd int, offset int64) int64 {
	nextHole, err := unix.Seek(fd, offset, unix.SEEK_HOLE)
	if err != nil {
		return -1
	}

	return nextHole
}

func findData(fd int, offset int64) int64 {
	nextHole, err := unix.Seek(fd, offset, unix.SEEK_DATA)
	if err != nil {
		return -1
	}

	return nextHole
}
