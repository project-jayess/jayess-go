package semantic

var builtinGlobals = []string{
	"childProcess",
	"compression",
	"console",
	"crypto",
	"decodeURI",
	"decodeURIComponent",
	"encodeURI",
	"encodeURIComponent",
	"dns",
	"fs",
	"globalThis",
	"http",
	"https",
	"llvm",
	"isFinite",
	"isNaN",
	"JSON",
	"Math",
	"os",
	"path",
	"parseFloat",
	"parseInt",
	"print",
	"process",
	"clearInterval",
	"clearTimeout",
	"queueMicrotask",
	"readKey",
	"readLine",
	"setInterval",
	"setTimeout",
	"storage",
	"sleep",
	"stream",
	"Symbol",
	"tcp",
	"terminal",
	"tls",
	"udp",
	"url",
	"util",
	"worker",
}

var builtinConstGlobals = []string{
	"Infinity",
	"NaN",
}

var builtinConstructableGlobals = []string{
	"AggregateError",
	"Array",
	"ArrayBuffer",
	"BigInt64Array",
	"BigUint64Array",
	"Buffer",
	"Date",
	"DataView",
	"EvalError",
	"Error",
	"Float32Array",
	"Float64Array",
	"Int8Array",
	"Int16Array",
	"Int32Array",
	"Map",
	"Object",
	"Promise",
	"RangeError",
	"ReferenceError",
	"RegExp",
	"Set",
	"SyntaxError",
	"Uint8Array",
	"Uint8ClampedArray",
	"Uint16Array",
	"Uint32Array",
	"TypeError",
	"URIError",
	"WeakMap",
	"WeakSet",
}

func newRootScope() *scope {
	root := newScope(nil)
	for _, name := range builtinGlobals {
		root.declareBuiltin(name)
	}
	for _, name := range builtinConstGlobals {
		root.declareBuiltinConst(name)
	}
	for _, name := range builtinConstructableGlobals {
		root.declareBuiltinConstructable(name)
	}
	return root
}
