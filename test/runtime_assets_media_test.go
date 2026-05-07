package test

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeAssetManifestLookupAndLoad(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "audio"), 0o755); err != nil {
		t.Fatalf("create audio dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "audio", "intro.wav"), []byte("wav-data"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	manifest, err := jayessruntime.AssetManifestFromJSON([]byte(`{
		"assets": [{"name":"intro","path":"audio/intro.wav","contentType":"audio/wav"}]
	}`))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	asset, ok := jayessruntime.AssetLookup(manifest, "intro")
	if !ok || asset.ContentType != "audio/wav" {
		t.Fatalf("unexpected asset lookup asset=%#v ok=%v", asset, ok)
	}
	data, err := jayessruntime.AssetLoad(root, manifest, "intro")
	if err != nil || string(data) != "wav-data" {
		t.Fatalf("load asset data=%q err=%v", data, err)
	}
	if _, err := jayessruntime.AssetLoad(root, manifest, "missing"); !errors.Is(err, jayessruntime.ErrAssetNotFound) {
		t.Fatalf("expected missing asset error, got %v", err)
	}
}

func TestRuntimeAudioParsesWAVMetadataAndMixesPCM(t *testing.T) {
	wav := makePCM16WAV(8000, 1, []int16{1000, 2000, -3000})
	audio, err := jayessruntime.AudioParseWAV(wav)
	if err != nil {
		t.Fatalf("parse wav: %v", err)
	}
	if audio.SampleRate != 8000 || audio.Channels != 1 || audio.BitsPerSample != 16 {
		t.Fatalf("unexpected wav metadata: %#v", audio)
	}
	metadata := jayessruntime.AudioMetadataFromPCM(audio)
	if metadata.FrameCount != 3 || metadata.DurationMS != 0 {
		t.Fatalf("unexpected audio metadata: %#v", metadata)
	}
	mixed := jayessruntime.AudioMixPCM(
		audio,
		jayessruntime.PCMAudio{SampleRate: 8000, Channels: 1, BitsPerSample: 16, Samples: []int16{32000, -1000, -32000}},
	)
	want := []int16{32767, 1000, -32768}
	for index, sample := range want {
		if mixed.Samples[index] != sample {
			t.Fatalf("mixed sample %d = %d, want %d", index, mixed.Samples[index], sample)
		}
	}
}

func TestRuntimeAudioQueueDrainsSeeksAppliesGainPanAndMixes(t *testing.T) {
	queue := jayessruntime.AudioNewQueue(48000, 2)
	jayessruntime.AudioQueuePush(queue, jayessruntime.PCMAudio{
		SampleRate:    48000,
		Channels:      2,
		BitsPerSample: 16,
		Samples:       []int16{1000, 2000, 3000, 4000, 5000, 6000},
	})
	first := jayessruntime.AudioQueueDrain(queue, 1)
	if len(first.Samples) != 2 || first.Samples[0] != 1000 || first.Samples[1] != 2000 {
		t.Fatalf("unexpected first drain: %#v", first.Samples)
	}

	jayessruntime.AudioQueueSeek(queue, 1)
	panned := jayessruntime.AudioQueueMix(queue, 1, jayessruntime.AudioMixControl{Gain: 0.5, Pan: 1})
	if len(panned.Samples) != 2 || panned.Samples[0] != 0 || panned.Samples[1] != 2000 {
		t.Fatalf("unexpected panned drain: %#v", panned.Samples)
	}

	mixed := jayessruntime.AudioMixPCM(
		jayessruntime.PCMAudio{SampleRate: 48000, Channels: 2, BitsPerSample: 16, Samples: []int16{30000, -30000}},
		jayessruntime.PCMAudio{SampleRate: 48000, Channels: 2, BitsPerSample: 16, Samples: []int16{10000, -10000}},
	)
	if len(mixed.Samples) != 2 || mixed.Samples[0] != 32767 || mixed.Samples[1] != -32768 {
		t.Fatalf("unexpected mixed samples: %#v", mixed.Samples)
	}
}

func makePCM16WAV(sampleRate int, channels int, samples []int16) []byte {
	dataSize := len(samples) * 2
	totalSize := 36 + dataSize
	wav := make([]byte, 44+dataSize)
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(totalSize))
	copy(wav[8:12], "WAVE")
	copy(wav[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1)
	binary.LittleEndian.PutUint16(wav[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(wav[24:28], uint32(sampleRate))
	byteRate := sampleRate * channels * 2
	binary.LittleEndian.PutUint32(wav[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(wav[32:34], uint16(channels*2))
	binary.LittleEndian.PutUint16(wav[34:36], 16)
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(dataSize))
	for index, sample := range samples {
		offset := 44 + index*2
		binary.LittleEndian.PutUint16(wav[offset:offset+2], uint16(sample))
	}
	return wav
}
