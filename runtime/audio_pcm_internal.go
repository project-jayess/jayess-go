package runtime

import "math"

type PCMAudio struct {
	SampleRate    int
	Channels      int
	BitsPerSample int
	Samples       []int16
}

type AudioMetadata struct {
	Format        string
	SampleRate    int
	Channels      int
	BitsPerSample int
	FrameCount    int
	DurationMS    int
}

func AudioMetadataFromPCM(audio PCMAudio) AudioMetadata {
	frameCount := 0
	if audio.Channels > 0 {
		frameCount = len(audio.Samples) / audio.Channels
	}
	durationMS := 0
	if audio.SampleRate > 0 {
		durationMS = int(math.Round(float64(frameCount) * 1000 / float64(audio.SampleRate)))
	}
	return AudioMetadata{
		Format:        "pcm_s16le",
		SampleRate:    audio.SampleRate,
		Channels:      audio.Channels,
		BitsPerSample: audio.BitsPerSample,
		FrameCount:    frameCount,
		DurationMS:    durationMS,
	}
}

func AudioMixPCM(inputs ...PCMAudio) PCMAudio {
	if len(inputs) == 0 {
		return PCMAudio{BitsPerSample: 16}
	}
	output := PCMAudio{
		SampleRate:    inputs[0].SampleRate,
		Channels:      inputs[0].Channels,
		BitsPerSample: 16,
		Samples:       make([]int16, maxSampleCount(inputs)),
	}
	for _, input := range inputs {
		for index, sample := range input.Samples {
			if index >= len(output.Samples) {
				break
			}
			output.Samples[index] = clampInt16(int(output.Samples[index]) + int(sample))
		}
	}
	return output
}

func maxSampleCount(inputs []PCMAudio) int {
	maxCount := 0
	for _, input := range inputs {
		if len(input.Samples) > maxCount {
			maxCount = len(input.Samples)
		}
	}
	return maxCount
}

func clampInt16(value int) int16 {
	if value > 32767 {
		return 32767
	}
	if value < -32768 {
		return -32768
	}
	return int16(value)
}
