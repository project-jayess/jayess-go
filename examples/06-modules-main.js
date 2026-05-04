import { add, twice } from "./modules/math.js";
import * as names from "./modules/names.js";

export const label = names.project;

function main(args) {
  return twice(add(2, 3)) === 10 && label === "jayess" ? 0 : 1;
}
