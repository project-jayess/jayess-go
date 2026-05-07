# Assets And Media

Jayess includes small internal helpers for project assets and simple audio data.
These helpers do not open audio devices and do not require SDL, OpenAL,
miniaudio, PortAudio, or platform audio libraries.

## Asset Manifest

An asset manifest maps stable names to packaged files:

```json
{
  "assets": [
    { "name": "intro", "path": "audio/intro.wav", "contentType": "audio/wav" }
  ]
}
```

The runtime can parse the manifest, look up an asset by name, and load the file
relative to a project or distribution root.

## WAV And PCM

The internal media helpers parse uncompressed 16-bit PCM WAV files into Jayess
runtime PCM data. They expose metadata such as sample rate, channel count, frame
count, and duration.

```js
function main() {
  const manifest = assets.parseManifest(fs.readFile("assets.json"));
  const wav = assets.load(".", manifest, "intro");
  const pcm = audio.parseWAV(wav);
  const info = audio.metadata(pcm);
  console.log(info.durationMS);
  return 0;
}
```

## Mixing

Simple PCM mixing sums matching sample positions and clamps to signed 16-bit
range. The internal queue supports appending PCM buffers, seeking by frame,
draining frames in order, and applying gain and stereo pan during mixing. This
is useful for asset processing and tests without opening an audio device.

```js
function main() {
  const queue = audio.queue(48000, 2);
  audio.push(queue, pcm);
  const next = audio.mix(audio.drain(queue, 512), { gain: 0.8, pan: -0.25 });
  console.log(next.samples.length);
  return 0;
}
```

Real-time playback, capture, device enumeration, and low-latency streaming
remain optional platform/native audio package responsibilities.
