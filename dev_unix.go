//go:build !windows

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func isBlockDevice(path string) bool {
	// Get the file information using Lstat
	fileInfo, err := os.Lstat(path)
	if err != nil {
		panic(err)
	}

	// Check if the mode is a device (block or character)
	if fileInfo.Mode()&os.ModeDevice != 0 {
		// Check if it's a block device specifically
		stat := fileInfo.Sys().(*syscall.Stat_t)
		if stat.Mode&syscall.S_IFBLK != 0 {
			return true
		}
	}

	return false
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

func buildSparseMap() {
	fd := int(reader.Fd())
	for pos := int64(0); pos < fileSize; {
		nextHole := findHole(fd, pos)
		if nextHole == -1 {
			break
		}
		nextData := findData(fd, nextHole)
		if nextData == -1 {
			break
		}
		sparseMap = append(sparseMap, Range{nextHole, nextData})
		pos = nextData
	}
	mapReady = true
}
