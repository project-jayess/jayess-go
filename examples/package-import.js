import double, { add as sum } from "@demo/math";

function main(args) {
  var value = sum(4, 5);
  print(double(value));
  return value;
}
