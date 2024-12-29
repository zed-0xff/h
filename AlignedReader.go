package main

import (
	"io"
	"os"
)

type AlignedReader struct {
	file  *os.File
	size  int64
	align int
}

func NewAlignedReader(file *os.File, size int64, align int) *AlignedReader {
	return &AlignedReader{file, size, align}
}

func (r *AlignedReader) ReadAt(buf []byte, offset int64) (n int, err error) {
	if offset >= r.size {
		return 0, io.EOF
	}

	// If offset or buffer size is not aligned, adjust
	if offset%int64(r.align) != 0 || len(buf)%r.align != 0 {
		// Calculate the aligned offset and the aligned buffer size
		alignedOffset := offset - offset%int64(r.align)

		// Aligned size must be at least the size of the buffer
		alignedSize := int64(len(buf)) + (offset - alignedOffset)

		// Adjust the aligned size to ensure it's a multiple of the alignment
		alignedSize += int64(r.align) - alignedSize%int64(r.align)

		if alignedOffset+alignedSize > r.size {
			alignedSize = r.size - alignedOffset
		}

		// Create a buffer large enough to hold the aligned data
		alignedBuf := make([]byte, alignedSize)

		// Read data into the aligned buffer starting at the aligned offset
		n, err = r.file.ReadAt(alignedBuf, alignedOffset)

		n -= int(offset - alignedOffset)
		if n > len(buf) {
			n = len(buf)
		}

		// Copy the relevant portion of the aligned buffer into the user's buffer
		copy(buf, alignedBuf[offset-alignedOffset:])

		return n, err
	}

	// If already aligned, just perform the read directly
	return r.file.ReadAt(buf, offset)
}

func (r *AlignedReader) Read(buf []byte) (n int, err error) {
	return r.file.Read(buf)
}

func (r *AlignedReader) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

func (r *AlignedReader) Fd() uintptr {
	return r.file.Fd()
}
