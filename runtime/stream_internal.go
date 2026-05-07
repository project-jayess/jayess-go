package runtime

import (
	"bytes"
	"io"
)

type TransformFunc func([]byte) ([]byte, error)

type StreamState struct {
	HighWaterMark int
	Buffered      int
}

func StreamReadable(data []byte) *IOStream {
	return NewReadableStream("readable", bytes.NewReader(data))
}

func StreamWritable() (*IOStream, *bytes.Buffer) {
	return NewBufferStream("writable")
}

func StreamDuplex(input []byte) (*IOStream, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	return NewDuplexStream("duplex", bytes.NewReader(input), buffer), buffer
}

func StreamTransform(source *IOStream, transform TransformFunc) (*IOStream, error) {
	data, err := source.ReadAll()
	if err != nil {
		return nil, err
	}
	if transform != nil {
		data, err = transform(data)
		if err != nil {
			return nil, err
		}
	}
	return NewReadableStream("transform", bytes.NewReader(data)), nil
}

func StreamAwaitDrain(state StreamState) bool {
	if state.HighWaterMark <= 0 {
		return true
	}
	return state.Buffered <= state.HighWaterMark
}

func StreamPipe(source *IOStream, sink *IOStream) (int64, error) {
	return PipeStream(source, sink)
}

func StreamReadAll(stream *IOStream) ([]byte, error) {
	if stream == nil {
		return nil, io.ErrClosedPipe
	}
	return stream.ReadAll()
}
