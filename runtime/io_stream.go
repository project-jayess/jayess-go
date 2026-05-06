package runtime

import (
	"bytes"
	"errors"
	"io"
)

type IOStream struct {
	name   string
	reader io.Reader
	writer io.Writer
	closer io.Closer
}

func NewReadableStream(name string, reader io.Reader) *IOStream {
	return &IOStream{name: name, reader: reader}
}

func NewWritableStream(name string, writer io.Writer) *IOStream {
	return &IOStream{name: name, writer: writer}
}

func NewDuplexStream(name string, reader io.Reader, writer io.Writer) *IOStream {
	return &IOStream{name: name, reader: reader, writer: writer}
}

func NewBufferStream(name string) (*IOStream, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	return NewDuplexStream(name, buffer, buffer), buffer
}

func (stream *IOStream) Name() string {
	if stream == nil {
		return ""
	}
	return stream.name
}

func (stream *IOStream) CanRead() bool {
	return stream != nil && stream.reader != nil
}

func (stream *IOStream) CanWrite() bool {
	return stream != nil && stream.writer != nil
}

func (stream *IOStream) ReadAll() ([]byte, error) {
	if stream == nil || stream.reader == nil {
		return nil, errors.New("stream is not readable")
	}
	return io.ReadAll(stream.reader)
}

func (stream *IOStream) Write(data []byte) (int, error) {
	if stream == nil || stream.writer == nil {
		return 0, errors.New("stream is not writable")
	}
	return stream.writer.Write(data)
}

func (stream *IOStream) WriteString(text string) (int, error) {
	return stream.Write([]byte(text))
}

func (stream *IOStream) Close() error {
	if stream == nil || stream.closer == nil {
		return nil
	}
	return stream.closer.Close()
}

func (stream *IOStream) PipeTo(sink *IOStream) (int64, error) {
	return PipeStream(stream, sink)
}

func PipeStream(source *IOStream, sink *IOStream) (int64, error) {
	if source == nil || source.reader == nil {
		return 0, errors.New("source stream is not readable")
	}
	if sink == nil || sink.writer == nil {
		return 0, errors.New("sink stream is not writable")
	}
	return io.Copy(sink.writer, source.reader)
}
