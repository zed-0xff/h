package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"strings"
	"syscall"
	"unsafe"
)

func getDeviceSize(drive string) (int64, error) {
	// Open a handle to the physical drive
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(drive),
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return 0, fmt.Errorf("error opening drive: %w", err)
	}
	defer windows.CloseHandle(handle)

	// IOCTL_DISK_GET_DRIVE_GEOMETRY_EX control code
	const IOCTL_DISK_GET_DRIVE_GEOMETRY_EX = 0x000700A0

	// Struct to receive disk size information
	type DiskGeometryEx struct {
		Geometry struct {
			Cylinders         int64
			MediaType         int32
			TracksPerCylinder uint32
			SectorsPerTrack   uint32
			BytesPerSector    uint32
		}
		DiskSize int64
	}

	var geometryEx DiskGeometryEx
	var bytesReturned uint32

	// Call DeviceIoControl to get the disk geometry
	err = windows.DeviceIoControl(
		handle,
		IOCTL_DISK_GET_DRIVE_GEOMETRY_EX,
		nil,
		0,
		(*byte)(unsafe.Pointer(&geometryEx)),
		uint32(unsafe.Sizeof(geometryEx)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error getting drive size: %w", err)
	}

	return geometryEx.DiskSize, nil
}

func isBlockDevice(fname string) bool {
	return strings.HasPrefix(fname, "\\\\.\\PhysicalDrive")
}

func getDeviceAlign(fname string) int {
	if isBlockDevice(fname) {
		return 512 // TODO: is it always 512 ?
	}
	return 0
}

const FSCTL_QUERY_ALLOCATED_RANGES = 0x940cf

type FILE_ALLOCATED_RANGE_BUFFER struct {
	FileOffset int64
	Length     int64
}

func buildSparseMap() {
	handle := syscall.Handle(reader.Fd())
	var input FILE_ALLOCATED_RANGE_BUFFER
	input.Length = fileSize

	var output [1024]FILE_ALLOCATED_RANGE_BUFFER
	var bytesReturned uint32

	err := syscall.DeviceIoControl(
		handle,
		FSCTL_QUERY_ALLOCATED_RANGES,
		(*byte)(unsafe.Pointer(&input)),
		uint32(unsafe.Sizeof(input)),
		(*byte)(unsafe.Pointer(&output[0])),
		uint32(len(output))*uint32(unsafe.Sizeof(output[0])),
		&bytesReturned,
		nil,
	)
	if err != nil {
		mapReady = false
		return
	}

	numRanges := int(bytesReturned) / int(unsafe.Sizeof(output[0]))
	var lastEnd int64 = 0

	// Iterate through allocated ranges to compute holes
	for i := 0; i < numRanges; i++ {
		allocated := output[i]
		if allocated.FileOffset > lastEnd {
			// There is a hole before this allocated range
			sparseMap = append(sparseMap, Range{lastEnd, allocated.FileOffset})
		}
		// Update the end of the last processed range
		lastEnd = allocated.FileOffset + allocated.Length
	}

	// Check for a hole at the end of the file
	if lastEnd < fileSize {
		sparseMap = append(sparseMap, Range{lastEnd, fileSize})
	}

	mapReady = true
}
