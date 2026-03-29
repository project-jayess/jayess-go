import current, { bump, counter, name as stateName } from "./lib/state.js";

function main(args) {
  print(stateName);
  print(counter);
  print(current());
  print(bump());
  print(counter);
  return counter;
}
