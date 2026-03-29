import { add, twice } from "./lib/math.js";

function main(args)
{
  var value = add(3, 4);
  print(twice(value));
  return value;
}
