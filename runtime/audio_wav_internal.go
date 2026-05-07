package runtime

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrUnsupportedAudioFormat = errors.New("unsupported audio format")

func AudioParseWAV(data []byte) (PCMAudio, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return PCMAudio{}, fmt.Errorf("%w: wav header", ErrUnsupportedAudioFormat)
	}
	var sampleRate int
	var channels int
	var bitsPerSample int
	var samples []int16
	for offset := 12; offset+8 <= len(data); {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			return PCMAudio{}, fmt.Errorf("%w: truncated chunk", ErrUnsupportedAudioFormat)
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return PCMAudio{}, fmt.Errorf("%w: fmt chunk", ErrUnsupportedAudioFormat)
			}
			format := binary.LittleEndian.Uint16(data[chunkStart : chunkStart+2])
			if format != 1 {
				return PCMAudio{}, fmt.Errorf("%w: non-pcm wav", ErrUnsupportedAudioFormat)
			}
			channels = int(binary.LittleEndian.Uint16(data[chunkStart+2 : chunkStart+4]))
			sampleRate = int(binary.LittleEndian.Uint32(data[chunkStart+4 : chunkStart+8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(data[chunkStart+14 : chunkStart+16]))
			if bitsPerSample != 16 {
				return PCMAudio{}, fmt.Errorf("%w: wav bits", ErrUnsupportedAudioFormat)
			}
		case "data":
			samples = decodePCM16LE(data[chunkStart:chunkEnd])
		}
		offset = chunkEnd
		if offset%2 == 1 {
			offset++
		}
	}
	if sampleRate == 0 || channels == 0 || len(samples) == 0 {
		return PCMAudio{}, fmt.Errorf("%w: incomplete wav", ErrUnsupportedAudioFormat)
	}
	return PCMAudio{
		SampleRate:    sampleRate,
		Channels:      channels,
		BitsPerSample: bitsPerSample,
		Samples:       samples,
	}, nil
}

func decodePCM16LE(data []byte) []int16 {
	samples := make([]int16, 0, len(data)/2)
	for offset := 0; offset+2 <= len(data); offset += 2 {
		samples = append(samples, int16(binary.LittleEndian.Uint16(data[offset:offset+2])))
	}
	return samples
}
