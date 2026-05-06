# Native Linking

Native bindings describe how C or C++ code is compiled and linked with Jayess
programs.

## Paths and Flags

- `sources` lists C/C++ source files for the wrapper or library
- `includeDirs` adds header search paths
- `libraryDirs` adds linker search paths
- `sharedLibraries` names libraries or prebuilt shared library files
- `cflags` and `ldflags` add compile and link flags
- `platforms` overrides these values per operating system

## Runtime Libraries

If a binding links to a shared library, the app distribution must include the
runtime library or document that the target system provides it.

## ABI

C++ wrappers should expose C ABI entrypoints with `extern "C"` so symbol names
are stable.

## Example Platform Flags

```js
export default bind({
  sources: ["./webview.cpp"],
  platforms: {
    linux: { ldflags: ["-lgtk-3", "-lwebkit2gtk-4.1"] },
    darwin: { ldflags: ["-framework", "Cocoa", "-framework", "WebKit"] },
    windows: { ldflags: ["-lole32", "-lcomctl32"] }
  },
  exports: {
    createWindow: { symbol: "jayess_webview_create_window", type: "function" }
  }
});
```
