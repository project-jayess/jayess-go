import { createModule, emitObject, link } from "llvm";

function main(args) {
  var api = createModule && emitObject && link;
  return api ? 0 : 1;
}
