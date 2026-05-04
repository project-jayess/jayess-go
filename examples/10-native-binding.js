import { bind } from "ffi";

const f = () => {};
export const add = f;
export const version = f;

export default bind({
  sources: ["./native/mylib.c"],
  includeDirs: ["./native/include"],
  libraryDirs: ["./native/lib"],
  sharedLibraries: ["mylib"],
  cflags: ["-DMYLIB_BINDING=1"],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" },
    version: { symbol: "mylib_version", type: "value" }
  }
});
