//go:build windows
package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

func getDriveSize(drive string) (int64, error) {
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
