package runtime

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrUnsupportedCompressionFormat = errors.New("unsupported compression format")

func CompressionGzip(data []byte) ([]byte, error) {
	var output bytes.Buffer
	writer := gzip.NewWriter(&output)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func CompressionGunzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func CompressionDeflate(data []byte) ([]byte, error) {
	var output bytes.Buffer
	writer, err := flate.NewWriter(&output, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func CompressionInflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()
	return io.ReadAll(reader)
}

func CompressionBrotliCompress(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: brotli", ErrUnsupportedCompressionFormat)
}

func CompressionBrotliDecompress(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: brotli", ErrUnsupportedCompressionFormat)
}

func CompressionCreateCompressStream(format string, source *IOStream) (*IOStream, error) {
	data, err := source.ReadAll()
	if err != nil {
		return nil, err
	}
	compressed, err := CompressionCompress(format, data)
	if err != nil {
		return nil, err
	}
	return NewReadableStream("compression-"+normalizeCompressionFormat(format), bytes.NewReader(compressed)), nil
}

func CompressionCreateDecompressStream(format string, source *IOStream) (*IOStream, error) {
	data, err := source.ReadAll()
	if err != nil {
		return nil, err
	}
	plain, err := CompressionDecompress(format, data)
	if err != nil {
		return nil, err
	}
	return NewReadableStream("decompression-"+normalizeCompressionFormat(format), bytes.NewReader(plain)), nil
}

func CompressionCompress(format string, data []byte) ([]byte, error) {
	switch normalizeCompressionFormat(format) {
	case "gzip", "gz":
		return CompressionGzip(data)
	case "deflate", "flate":
		return CompressionDeflate(data)
	case "brotli", "br":
		return CompressionBrotliCompress(data)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCompressionFormat, format)
	}
}

func CompressionDecompress(format string, data []byte) ([]byte, error) {
	switch normalizeCompressionFormat(format) {
	case "gzip", "gz":
		return CompressionGunzip(data)
	case "deflate", "flate":
		return CompressionInflate(data)
	case "brotli", "br":
		return CompressionBrotliDecompress(data)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCompressionFormat, format)
	}
}

func normalizeCompressionFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}
