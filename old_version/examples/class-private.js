import { BaseCounter, DeepCounter } from "./lib/class-private.js";

function main(args) {
  var counter = new DeepCounter(2, 3);
  var base = new BaseCounter(4);
  print(DeepCounter.familyName());
  print(DeepCounter.shadow());
  print(counter.label);
  print(counter.total());
  print(counter.reveal());
  print(base.readSecret());
  return counter.total();
}
