package runtime

import "math"

type AudioQueue struct {
	SampleRate    int
	Channels      int
	BitsPerSample int
	samples       []int16
	cursor        int
}

type AudioMixControl struct {
	Gain float64
	Pan  float64
}

func AudioNewQueue(sampleRate int, channels int) *AudioQueue {
	if channels <= 0 {
		channels = 1
	}
	return &AudioQueue{SampleRate: sampleRate, Channels: channels, BitsPerSample: 16}
}

func AudioQueuePush(queue *AudioQueue, audio PCMAudio) {
	if queue == nil {
		return
	}
	if queue.SampleRate == 0 {
		queue.SampleRate = audio.SampleRate
	}
	if queue.Channels == 0 {
		queue.Channels = audio.Channels
	}
	queue.samples = append(queue.samples, audio.Samples...)
}

func AudioQueueSeek(queue *AudioQueue, frameOffset int) {
	if queue == nil {
		return
	}
	if frameOffset < 0 {
		frameOffset = 0
	}
	sampleOffset := frameOffset * queue.Channels
	if sampleOffset > len(queue.samples) {
		sampleOffset = len(queue.samples)
	}
	queue.cursor = sampleOffset
}

func AudioQueueDrain(queue *AudioQueue, frameCount int) PCMAudio {
	if queue == nil || frameCount <= 0 {
		return PCMAudio{BitsPerSample: 16}
	}
	sampleCount := frameCount * queue.Channels
	end := queue.cursor + sampleCount
	if end > len(queue.samples) {
		end = len(queue.samples)
	}
	drained := append([]int16(nil), queue.samples[queue.cursor:end]...)
	queue.cursor = end
	return PCMAudio{
		SampleRate:    queue.SampleRate,
		Channels:      queue.Channels,
		BitsPerSample: 16,
		Samples:       drained,
	}
}

func AudioQueueMix(queue *AudioQueue, frameCount int, controls AudioMixControl) PCMAudio {
	return AudioApplyGainPan(AudioQueueDrain(queue, frameCount), controls)
}

func AudioApplyGainPan(audio PCMAudio, controls AudioMixControl) PCMAudio {
	gain := controls.Gain
	if gain == 0 {
		gain = 1
	}
	pan := math.Max(-1, math.Min(1, controls.Pan))
	output := PCMAudio{
		SampleRate:    audio.SampleRate,
		Channels:      audio.Channels,
		BitsPerSample: 16,
		Samples:       make([]int16, len(audio.Samples)),
	}
	if audio.Channels == 2 {
		leftGain := gain
		rightGain := gain
		if pan < 0 {
			rightGain *= 1 + pan
		} else if pan > 0 {
			leftGain *= 1 - pan
		}
		for index, sample := range audio.Samples {
			channelGain := leftGain
			if index%2 == 1 {
				channelGain = rightGain
			}
			output.Samples[index] = clampInt16(int(math.Round(float64(sample) * channelGain)))
		}
		return output
	}
	for index, sample := range audio.Samples {
		output.Samples[index] = clampInt16(int(math.Round(float64(sample) * gain)))
	}
	return output
}
