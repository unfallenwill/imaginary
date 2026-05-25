//go:build cgo && ffmpeg

package avio

import (
	"fmt"
	"io"

	"github.com/asticode/go-astiav"
)

const ioBufferSize = 32768

// NewMemoryIOContext creates an astiav.IOContext that reads from a byte slice
// with full seek support. The caller must call Free() on the returned context.
func NewMemoryIOContext(data []byte) (*astiav.IOContext, error) {
	offset := int64(0)
	size := int64(len(data))

	readFunc := func(b []byte) (int, error) {
		if offset >= size {
			return 0, io.EOF
		}
		n := copy(b, data[offset:])
		offset += int64(n)
		return n, nil
	}

	seekFunc := func(off int64, whence int) (int64, error) {
		switch whence {
		case io.SeekStart:
			offset = off
		case io.SeekCurrent:
			offset += off
		case io.SeekEnd:
			offset = size + off
		default:
			return 0, fmt.Errorf("unsupported whence: %d", whence)
		}
		if offset < 0 {
			return 0, fmt.Errorf("negative seek position %d", offset)
		}
		return offset, nil
	}

	return astiav.AllocIOContext(ioBufferSize, false, readFunc, seekFunc, nil)
}
