package runtime

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Buffer struct {
	Data []byte
}

func BufferCreate(size int) *Buffer {
	if size < 0 {
		size = 0
	}
	return &Buffer{Data: make([]byte, size)}
}

func BufferFromString(text string, encoding string) (*Buffer, error) {
	if encoding != "" && encoding != "utf8" && encoding != "utf-8" {
		return nil, fmt.Errorf("unsupported buffer encoding %q", encoding)
	}
	return &Buffer{Data: []byte(text)}, nil
}

func BufferToString(buffer *Buffer, encoding string) (string, error) {
	if encoding != "" && encoding != "utf8" && encoding != "utf-8" {
		return "", fmt.Errorf("unsupported buffer encoding %q", encoding)
	}
	if buffer == nil {
		return "", nil
	}
	return string(buffer.Data), nil
}

func BufferSlice(buffer *Buffer, start int, end int) *Buffer {
	if buffer == nil {
		return BufferCreate(0)
	}
	start, end = clampRange(len(buffer.Data), start, end)
	return &Buffer{Data: append([]byte(nil), buffer.Data[start:end]...)}
}

func BufferCopy(source *Buffer, target *Buffer, offset int) int {
	if source == nil || target == nil || offset < 0 || offset >= len(target.Data) {
		return 0
	}
	return copy(target.Data[offset:], source.Data)
}

func BufferReadUInt16LE(buffer *Buffer, offset int) (uint16, error) {
	if buffer == nil || offset < 0 || offset+2 > len(buffer.Data) {
		return 0, fmt.Errorf("buffer read out of range")
	}
	return binary.LittleEndian.Uint16(buffer.Data[offset : offset+2]), nil
}

func BufferWriteUInt16LE(buffer *Buffer, value uint16, offset int) error {
	if buffer == nil || offset < 0 || offset+2 > len(buffer.Data) {
		return fmt.Errorf("buffer write out of range")
	}
	binary.LittleEndian.PutUint16(buffer.Data[offset:offset+2], value)
	return nil
}

func BufferTypedArrayView(buffer *Buffer, view string) ([]byte, error) {
	if view != "" && view != "Uint8Array" {
		return nil, fmt.Errorf("unsupported typed array view %q", view)
	}
	if buffer == nil {
		return nil, nil
	}
	return buffer.Data, nil
}

func BufferCreateReadStream(buffer *Buffer) *IOStream {
	if buffer == nil {
		return NewReadableStream("buffer-read", bytes.NewReader(nil))
	}
	return NewReadableStream("buffer-read", bytes.NewReader(buffer.Data))
}

func BufferCreateWriteStream(buffer *Buffer) *IOStream {
	if buffer == nil {
		buffer = BufferCreate(0)
	}
	writer := &bufferWriter{buffer: buffer}
	return NewWritableStream("buffer-write", writer)
}

type bufferWriter struct {
	buffer *Buffer
}

func (writer *bufferWriter) Write(data []byte) (int, error) {
	if writer == nil || writer.buffer == nil {
		return 0, fmt.Errorf("buffer writer is closed")
	}
	writer.buffer.Data = append(writer.buffer.Data, data...)
	return len(data), nil
}

func clampRange(length int, start int, end int) (int, int) {
	if start < 0 {
		start = 0
	}
	if end < 0 || end > length {
		end = length
	}
	if start > length {
		start = length
	}
	if end < start {
		end = start
	}
	return start, end
}
