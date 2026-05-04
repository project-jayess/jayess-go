
import { loadImage, loadImageFromBytes, loadTexture } from "@jayess/raylib";

function main(args) {
  try {
    loadImage("missing-image.png");
  } catch (err) {
    console.log("raylib-image-error:" + err.name);
  }
  try {
    loadImageFromBytes(".ppm", Uint8Array.fromString("not-an-image"));
  } catch (err) {
    console.log("raylib-bytes-error:" + err.name);
  }
  try {
    loadTexture("missing-texture.png");
  } catch (err) {
    console.log("raylib-texture-error:" + err.name);
  }
  return 0;
}
