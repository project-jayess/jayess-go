import { VERSION as version, label as rename } from "./lib/meta.js";

function main(args) {
  print(version);
  print(rename("jayess"));
  return 0;
}
