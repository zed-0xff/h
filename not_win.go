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
