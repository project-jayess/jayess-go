import { readFile } from "fs";
import * as path from "path";
import { createModule } from "llvm";

function main(args) {
  var name = path ? "available" : "missing";
  return readFile && createModule && name === "available" ? 0 : 1;
}
