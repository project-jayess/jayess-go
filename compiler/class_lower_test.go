package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestCompileParsesClassWithoutTranspilerFormattingAssumptions(t *testing.T) {
	source := `
class Counter
{
  value = 1;

  constructor(step)
  {
    this.step = step;
  }

  total()
  {
    return this.value + this.step;
  }
}

function main(args) {
  var counter = new Counter(2);
  return counter.total();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.LLVMIR) == 0 {
		t.Fatalf("expected LLVM IR output")
	}
}

func TestCompileSupportsDerivedClassSuperAndPrivateFields(t *testing.T) {
	source := `
class Base {
  value = 2;

  read() {
    return this.value;
  }
}

class Child extends Base {
  #bonus = 3;

  constructor(multiplier) {
    super();
    this.multiplier = multiplier;
  }

  total() {
    return super.read() + this.#bonus + this.multiplier;
  }
}

function main(args) {
  var child = new Child(4);
  return child.total();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.LLVMIR) == 0 {
		t.Fatalf("expected LLVM IR output")
	}
}

func TestCompileRejectsPrivateFieldAccessOutsideDeclaringClass(t *testing.T) {
	source := `
class Counter {
  #secret = 2;
}

function main(args) {
  var counter = new Counter();
  return counter.#secret;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected Compile to reject private field access")
	}
	if !strings.Contains(err.Error(), "private fields are only accessible") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileUsesInstanceDispatchHelpersForMethodCalls(t *testing.T) {
	source := `
class Animal {
  sound() {
    return "animal";
  }
}

class Dog extends Animal {
  sound() {
    return "dog";
  }
}

function main(args) {
  var dog = new Dog();
  print(dog.sound());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@__jayess_dispatch__sound__0") {
		t.Fatalf("expected dispatch helper in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_value_from_function") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected direct method calls to flow through first-class function values, got:\n%s", irText)
	}
	if !strings.Contains(irText, "__jayess_class") {
		t.Fatalf("expected hidden class tag access in LLVM IR, got:\n%s", irText)
	}
}

func TestCompileSupportsExtractedInstanceMethodCalls(t *testing.T) {
	source := `
class Dog {
  sound() {
    return "dog";
  }
}

function main(args) {
  var dog = new Dog();
  var speak = dog.sound;
  print(speak());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@__jayess_dispatch__sound__0") {
		t.Fatalf("expected extracted instance method to use dispatch helper")
	}
	if !strings.Contains(irText, "@jayess_value_from_function") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected extracted instance method to flow through first-class function values")
	}
}

func TestCompileSupportsExtractedStaticMethodCalls(t *testing.T) {
	source := `
class Dog {
  static kind() {
    return "dog";
  }
}

function main(args) {
  var kind = Dog.kind;
  print(kind());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@Dog__kind") {
		t.Fatalf("expected extracted static method to call lowered static symbol")
	}
	if !strings.Contains(irText, "@jayess_value_from_function") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected extracted static method to flow through first-class function values")
	}
}

func TestCompileSupportsMapStandardLibrarySurface(t *testing.T) {
	source := `
function main(args) {
  var map = new Map();
  map.set("name", "kimchi");
  print(map.get("name"));
  print(map.has("name"));
  print(map.size);
  print(map.keys());
  print(map.values());
  print(map.entries());
  map.clear();
  map.delete("name");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_map_new") {
		t.Fatalf("expected Map constructor lowering in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected Map method/property access through runtime member lookup, got:\n%s", irText)
	}
}

func TestCompileSupportsSetDateAndJSONStandardLibrarySurface(t *testing.T) {
	source := `
function main(args) {
  var set = new Set();
  set.add("kimchi");
  print(set.has("kimchi"));
  print(set.size);
  print(set.values());
  print(set.entries());
  set.clear();

  var now = Date.now();
  var date = new Date(now);
  print(date.getTime());
  print(date.toString());
  print(date.toISOString());

  var text = JSON.stringify({ name: "kimchi", spicy: true });
  var data = JSON.parse(text);
  print(data.name);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_std_set_new",
		"@jayess_std_date_now",
		"@jayess_std_date_new",
		"@jayess_std_json_stringify",
		"@jayess_std_json_parse",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
	}
}

func TestCompileSupportsMapSetIteratorsForForOf(t *testing.T) {
	source := `
function main(args) {
  var map = new Map();
  map.set("name", "kimchi");
  map.set("kind", "jjigae");
  for (var entry of map) {
    print(entry[0], entry[1]);
  }

  var set = new Set();
  set.add("kimchi");
  set.add("jjigae");
  for (var value of set) {
    print(value);
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_iterable_values") {
		t.Fatalf("expected for...of lowering to use iterable helper, got:\n%s", irText)
	}
}

func TestCompileSupportsMathObjectStringAndArrayHelperSurface(t *testing.T) {
	source := `
function main(args) {
  var text = "  kimchi jjigae  ";
  print(text.length);
  print(text.includes("kimchi"));
  print(text.startsWith("  kim"));
  print(text.endsWith("e  "));
  print(text.slice(2, 8));
  print(text.trim());
  print(text.toUpperCase());
  print(text.toLowerCase());
  print(text.split(" "));

  var obj = { name: "kimchi", spicy: 10 };
  print(Object.keys(obj));
  print(Object.values(obj));
  print(Object.entries(obj));
  print(Object.hasOwn(obj, "name"));
  print(Object.assign({}, obj));

  var values = [1, 2, 3];
  print(values.includes(2));
  print(values.join("-"));
  print(values.map((x) => x * 2));
  print(values.filter((x) => x > 1));
  print(values.find((x) => x == 2));

  print(Math.floor(1.8));
  print(Math.ceil(1.2));
  print(Math.round(1.5));
  print(Math.min(1, 2));
  print(Math.max(1, 2));
  print(Math.abs(-2));
  print(Math.pow(2, 3));
  print(Math.sqrt(9));
  print(Math.random());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_value_get_member",
		"@jayess_value_object_values",
		"@jayess_value_object_entries",
		"@jayess_value_object_assign",
		"@jayess_value_object_has_own",
		"@jayess_math_floor",
		"@jayess_math_pow",
		"@jayess_math_random",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
	}
}

func TestCompileSupportsRegExpAndStringRegexHelpers(t *testing.T) {
	source := `
function main(args) {
  var re = new RegExp("kim.+");
  print(re.source);
  print(re.test("kimchi"));
  var text = "kimchi jjigae";
  print(text.match(re));
  print(text.search(re));
  print(text.replace(re, "food"));
  print(text.split(re));
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_regexp_new") {
		t.Fatalf("expected RegExp constructor lowering in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected regex/string helpers to use runtime member lookup, got:\n%s", irText)
	}
}

func TestCompileSupportsStaticHelpersAndConsoleSurface(t *testing.T) {
	source := `
function main(args) {
  console.log("hello", 1);
  console.warn("warn");
  console.error("error");

  print(Number.isNaN(0 / 0));
  print(Number.isFinite(10));
  print(String.fromCharCode(65, 66));
  print(Array.isArray([1, 2]));
  print(Array.from(new Set()));
  print(Array.of(1, 2, 3));
  print(Object.fromEntries([["name", "kimchi"], ["spicy", 10]]));
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_console_log",
		"@jayess_console_warn",
		"@jayess_console_error",
		"@jayess_std_number_is_nan",
		"@jayess_std_number_is_finite",
		"@jayess_std_string_from_char_code",
		"@jayess_std_array_is_array",
		"@jayess_std_array_from",
		"@jayess_std_array_of",
		"@jayess_std_object_from_entries",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
	}
}

func TestCompileSupportsErrorBinaryIteratorAwaitAndTypeAnnotations(t *testing.T) {
	source := `
function add(left: number, right: number): number {
  return left + right;
}

function main(args: array): number {
  var total: number = add(1, 2);
  total = total + 1;
  var err = new Error("boom");
  var typed = new TypeError("bad");
  console.log(err.name, err.message, err.toString(), typed.toString());

  var buffer = new ArrayBuffer(4);
  var bytes = new Uint8Array(buffer);
  bytes[0] = 300;
  bytes.fill(7);
  console.log(buffer.byteLength, bytes.length, bytes[0]);

  var iter = Iterator.from([1, 2]);
  console.log(iter.next().value, iter.next().done);

  var value = await Promise.resolve(total);
  console.log(value);
  var chained = Promise.resolve(2).then((x) => x + 3);
  console.log(await chained);
  try {
    await Promise.reject(new Error("nope"));
  } catch (caught) {
    console.log(caught.message);
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_std_error_new",
		"@jayess_std_array_buffer_new",
		"@jayess_std_uint8_array_new",
		"@jayess_std_iterator_from",
		"@jayess_std_promise_resolve",
		"@jayess_std_promise_reject",
		"@jayess_await",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
	}
}

func TestCompileEnforcesOptionalTypeAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "variable initializer",
			source: `
function main(args) {
  var count: number = "kimchi";
  return 0;
}
`,
			message: "cannot initialize number variable count with string",
		},
		{
			name: "assignment",
			source: `
function main(args) {
  var count: number = 1;
  count = "kimchi";
  return 0;
}
`,
			message: "cannot assign string to number",
		},
		{
			name: "call argument",
			source: `
function add(left: number, right: number): number {
  return left + right;
}

function main(args) {
  console.log(add("kimchi", 2));
  return 0;
}
`,
			message: "argument 1 for add expects number, got string",
		},
		{
			name: "return",
			source: `
function label(): string {
  return 10;
}

function main(args) {
  console.log(label());
  return 0;
}
`,
			message: "function label must return string, got number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
			if err == nil {
				t.Fatalf("expected compile error")
			}
			if !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected %q in error, got %v", tt.message, err)
			}
		})
	}
}

func TestCompileKeepsTypeAnnotationsOptional(t *testing.T) {
	source := `
function main(args) {
  var value = 1;
  value = "kimchi";
  console.log(value);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileWarnsWhenUsingDeprecatedPrint(t *testing.T) {
	source := `
function main(args) {
  print("hello");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected deprecation warning for print")
	}
	if !strings.Contains(result.Warnings[0].Message, "deprecated") {
		t.Fatalf("unexpected warning: %v", result.Warnings)
	}
	if result.Warnings[0].Line != 3 || result.Warnings[0].Column != 3 {
		t.Fatalf("expected warning source span 3:3, got %d:%d", result.Warnings[0].Line, result.Warnings[0].Column)
	}
}

func TestCompileWarnsWhenUsingGlobalTimerAliases(t *testing.T) {
	source := `
function main(args) {
  sleepAsync(1, "done");
  var id = setTimeout(() => {
    return 0;
  }, 1);
  clearTimeout(id);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.Warnings) != 3 {
		t.Fatalf("expected 3 timer compatibility warnings, got %d: %#v", len(result.Warnings), result.Warnings)
	}
	expected := []struct {
		name   string
		line   int
		column int
		use    string
	}{
		{name: "sleepAsync", line: 3, column: 3, use: "timers.sleep"},
		{name: "setTimeout", line: 4, column: 12, use: "timers.setTimeout"},
		{name: "clearTimeout", line: 7, column: 3, use: "timers.clearTimeout"},
	}
	for i, want := range expected {
		warning := result.Warnings[i]
		if warning.Code != "JY001" || warning.Category != "deprecation" {
			t.Fatalf("expected deprecation warning JY001, got %#v", warning)
		}
		if !strings.Contains(warning.Message, want.name) || !strings.Contains(warning.Message, want.use) {
			t.Fatalf("expected warning to mention %q and %q, got %q", want.name, want.use, warning.Message)
		}
		if warning.Line != want.line || warning.Column != want.column {
			t.Fatalf("expected %s warning span %d:%d, got %d:%d", want.name, want.line, want.column, warning.Line, warning.Column)
		}
	}
}

func TestCompileWarningPolicyNoneSuppressesWarnings(t *testing.T) {
	source := `
function main(args) {
  print("hello");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc", WarningPolicy: "none"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected warning policy none to suppress warnings, got %#v", result.Warnings)
	}
}

func TestCompileWarningPolicyErrorFailsOnWarning(t *testing.T) {
	source := `
function main(args) {
  print("hello");
  return 0;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc", WarningPolicy: "error"})
	if err == nil {
		t.Fatalf("expected warning policy error to fail")
	}
	compileErr, ok := err.(*CompileError)
	if !ok {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Severity != "error" || compileErr.Diagnostic.Code != "JY001" {
		t.Fatalf("expected escalated JY001 error, got %#v", compileErr.Diagnostic)
	}
	if len(compileErr.Diagnostic.Notes) == 0 || !strings.Contains(compileErr.Diagnostic.Notes[0], "warnings are treated as errors") {
		t.Fatalf("expected warning policy note, got %#v", compileErr.Diagnostic.Notes)
	}
}

func TestCompileWarningPolicyErrorAllowsConfiguredCategory(t *testing.T) {
	source := `
function main(args) {
  print("hello");
  return 0;
}
`

	result, err := Compile(source, Options{
		TargetTriple:             "x86_64-pc-windows-msvc",
		WarningPolicy:            "error",
		AllowedWarningCategories: []string{"deprecation"},
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected allowed deprecation warning to remain visible, got %#v", result.Warnings)
	}
	if result.Warnings[0].Category != "deprecation" {
		t.Fatalf("expected deprecation warning, got %#v", result.Warnings[0])
	}
}

func TestCompileRejectsUnknownWarningPolicy(t *testing.T) {
	source := `
function main(args) {
  return 0;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc", WarningPolicy: "invalid"})
	if err == nil || !strings.Contains(err.Error(), "unsupported warning policy") {
		t.Fatalf("expected unsupported warning policy error, got: %v", err)
	}
}

func TestCompilePathReportsSemanticSourceSpan(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.js")
	source := "function main(args) {\n  return missing;\n}\n"
	if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(input, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	text := err.Error()
	if !strings.Contains(text, input+":2:10:") {
		t.Fatalf("expected semantic error with file and source span, got: %s", text)
	}
	if !strings.Contains(text, "unknown identifier missing") {
		t.Fatalf("expected semantic error message, got: %s", text)
	}
}

func TestCompileSupportsProcessPathAndFsSurface(t *testing.T) {
	source := `
function main(args) {
  var cwd = process.cwd();
  var argv = process.argv();
  var platform = process.platform();
  var arch = process.arch();
  var home = process.env("HOME");
  var file = path.join("build", "tmp.txt");
  var sep = path.sep;
  var delimiter = path.delimiter;
  var normalized = path.normalize("build/./tmp.txt");
  var resolved = path.resolve("build", "tmp.txt");
  var relative = path.relative("build", file);
  var parsed = path.parse(file);
  var formatted = path.format(parsed);
  var absolute = path.isAbsolute(file);
  var base = path.basename(file);
  var dir = path.dirname(file);
  var ext = path.extname(file);
  var wrote = fs.writeFile(file, "kimchi");
  var made = fs.mkdir(path.join("build", "tmpdir"), { recursive: true });
  var exists = fs.exists(file);
  var text = fs.readFile(file, "utf8");
  var stat = fs.stat(file);
  var entries = fs.readDir("build", { recursive: true });
  var copied = fs.copyFile(file, path.join("build", "tmp-copy.txt"));
  var copiedDir = fs.copyDir("build", "build-copy");
  var renamed = fs.rename(path.join("build", "tmp-copy.txt"), path.join("build", "tmp-copy-2.txt"));
  var removed = fs.remove("build-copy", { recursive: true });
  console.log(cwd, argv, platform, arch, home, sep, delimiter, normalized, resolved, relative, parsed, formatted, absolute, base, dir, ext, wrote, made, exists, text, stat, entries, copied, copiedDir, renamed, removed);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_std_process_cwd",
		"@jayess_std_process_argv",
		"@jayess_std_process_platform",
		"@jayess_std_process_arch",
		"@jayess_std_process_env",
		"@jayess_std_path_join",
		"@jayess_std_path_normalize",
		"@jayess_std_path_resolve",
		"@jayess_std_path_relative",
		"@jayess_std_path_parse",
		"@jayess_std_path_format",
		"@jayess_std_path_sep",
		"@jayess_std_path_delimiter",
		"@jayess_std_path_is_absolute",
		"@jayess_std_path_basename",
		"@jayess_std_path_dirname",
		"@jayess_std_path_extname",
		"@jayess_std_fs_read_file",
		"@jayess_std_fs_write_file",
		"@jayess_std_fs_exists",
		"@jayess_std_fs_read_dir",
		"@jayess_std_fs_stat",
		"@jayess_std_fs_mkdir",
		"@jayess_std_fs_remove",
		"@jayess_std_fs_copy_file",
		"@jayess_std_fs_copy_dir",
		"@jayess_std_fs_rename",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
	}
}

func TestCompileSupportsProcessExit(t *testing.T) {
	source := `
function main(args) {
  process.exit(0);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_std_process_exit") {
		t.Fatalf("expected process exit runtime symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompileSupportsNamedFunctionReferences(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  var fn = greet;
  var alias = fn;
  print(alias());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@greet(") {
		t.Fatalf("expected named function reference call to lower to greet")
	}
}

func TestCompileSupportsDynamicVarReassignmentAcrossValueKinds(t *testing.T) {
	source := `
function main(args) {
  var value = 1;
  value = "kimchi";
  value = {};
  value.name = "jjigae";
  print(value.name);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "call ptr @jayess_value_from_number") {
		t.Fatalf("expected var initialization to box number values, got:\n%s", irText)
	}
	if !strings.Contains(irText, "call ptr @jayess_value_from_string") {
		t.Fatalf("expected var reassignment to box string values, got:\n%s", irText)
	}
}

func TestCompileSupportsStringIndexedObjectMutationAndDelete(t *testing.T) {
	source := `
function main(args) {
  var kimchi = {};
  kimchi["name"] = "kimchi";
  print(kimchi["name"]);
  delete kimchi["name"];
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_object_set_value") {
		t.Fatalf("expected object string-index assignment in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_object_get") {
		t.Fatalf("expected object string-index read in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_object_delete") {
		t.Fatalf("expected object delete in LLVM IR, got:\n%s", irText)
	}
}

func TestCompileSupportsFunctionExpressions(t *testing.T) {
	source := `
function main(args) {
  var add = function(a, b) {
    return a + b;
  };
  return add(2, 3);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@__jayess_lambda_0(") {
		t.Fatalf("expected lowered function expression helper in LLVM IR")
	}
}

func TestCompileSupportsArrowFunctions(t *testing.T) {
	source := `
function main(args) {
  var twice = (value) => value * 2;
  var alias = twice;
  return alias(4);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@__jayess_lambda_0(") {
		t.Fatalf("expected lowered arrow function helper in LLVM IR")
	}
}

func TestCompileSupportsArraySpreadLiterals(t *testing.T) {
	source := `
function main(args) {
  var tail = [2, 3];
  var values = [1, ...tail];
  return values.length;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_array_append_array") {
		t.Fatalf("expected array spread lowering in LLVM IR, got:\n%s", irText)
	}
}

func TestCompileSupportsSpreadCallsForNamedFunctions(t *testing.T) {
	source := `
function add(a, b) {
  return a + b;
}

function main(args) {
  var values = [1, 2];
  return add(...values);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_array_append_array") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected spread call to lower through expanded array + apply path, got:\n%s", irText)
	}
}

func TestCompilePacksRestParametersForNamedFunctions(t *testing.T) {
	source := `
function count(head, ...tail) {
  return tail.length;
}

function main(args) {
  return count(1, 2, 3);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_array_new") || !strings.Contains(irText, "@jayess_array_push_value") {
		t.Fatalf("expected rest arguments to be packed into an array, got:\n%s", irText)
	}
}

func TestCompileSupportsSpreadIntoRestParameterFunctions(t *testing.T) {
	source := `
function count(head, ...tail) {
  return tail.length;
}

function main(args) {
  var extra = [2, 3];
  return count(1, ...extra);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_function_has_rest") || !strings.Contains(irText, "@jayess_array_append_array") {
		t.Fatalf("expected spread-into-rest call to use apply metadata + array expansion, got:\n%s", irText)
	}
}

func TestCompileSupportsTryCatchFinally(t *testing.T) {
	source := `
function main(args) {
  try {
    throw "boom";
  } catch (err) {
    print(err);
  } finally {
    print("done");
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_throw") || !strings.Contains(irText, "@jayess_take_exception") {
		t.Fatalf("expected try/catch/finally runtime hooks in LLVM IR, got:\n%s", irText)
	}
}

func TestCompileSupportsTryFinallyWithoutCatch(t *testing.T) {
	source := `
function main(args) {
  try {
    print("work");
  } finally {
    print("cleanup");
  }
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileSupportsStringConcatenation(t *testing.T) {
	source := `
function main(args) {
  print("asdas" + "asdsadsa");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_concat_values") {
		t.Fatalf("expected string concatenation to lower through runtime concat helper")
	}
}

func TestCompileSupportsMultiArgumentPrint(t *testing.T) {
	source := `
function main(args) {
  print("asdasdas", "asdasdsasda");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_print_many") {
		t.Fatalf("expected multi-argument print to use runtime print_many")
	}
}

func TestCompileSupportsStrictEqualityOperators(t *testing.T) {
	source := `
function main(args) {
  print(1 === 1);
  print(1 !== 2);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "fcmp oeq") || !strings.Contains(string(result.LLVMIR), "fcmp one") {
		t.Fatalf("expected strict equality operators to lower into comparison predicates")
	}
}

func TestCompileSupportsNullishCoalescing(t *testing.T) {
	source := `
function main(args) {
  var value = undefined;
  print(value ?? "fallback");
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_is_nullish") {
		t.Fatalf("expected nullish coalescing to use runtime nullish checks")
	}
}

func TestCompileSupportsOptionalChaining(t *testing.T) {
	source := `
function main(args) {
  var obj = null;
  print(obj?.name);
  print(obj?.["name"]);
  print(obj?.call?.());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_is_nullish") {
		t.Fatalf("expected optional chaining to use runtime nullish checks")
	}
}

func TestCompileSupportsDefaultParameters(t *testing.T) {
	source := `
function greet(name = "kimchi") {
  print(name);
  return 0;
}

function main(args) {
  return greet();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_is_undefined") && !strings.Contains(irText, "undefined") {
		t.Fatalf("expected default parameters to lower through undefined checks")
	}
}

func TestCompileSupportsObjectDestructuringDeclaration(t *testing.T) {
	source := `
function main(args) {
  const { a, b: renamed } = { a: 1, b: 2 };
  return a + renamed;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected object destructuring to lower through member access helpers")
	}
}

func TestCompileSupportsArrayDestructuringDeclarationAndAssignment(t *testing.T) {
	source := `
function main(args) {
  var first = 0;
  var third = 0;
  const [a, , c] = [1, 2, 3];
  [first, third] = [a, c];
  return first + third;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected array destructuring to lower through indexed access helpers")
	}
}

func TestCompileSupportsNestedDestructuring(t *testing.T) {
	source := `
function main(args) {
  const { point: { x }, values: [first] } = { point: { x: 4 }, values: [3, 2] };
  return x + first;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_member") || !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected nested destructuring to lower through nested member/index access")
	}
}

func TestCompileSupportsParameterDestructuring(t *testing.T) {
	source := `
function total({ a, b }, [x, y]) {
  return a + b + x + y;
}

function main(args) {
  return total({ a: 1, b: 2 }, [3, 4]);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_member") || !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected parameter destructuring to lower through member/index helpers")
	}
}

func TestCompileSupportsDefaultedDestructuredParameters(t *testing.T) {
	source := `
function greet({ name } = { name: "kimchi" }) {
  print(name);
  return 0;
}

function main(args) {
  return greet();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_member") || !strings.Contains(irText, "undefined") {
		t.Fatalf("expected defaulted destructured parameters to keep default-parameter lowering and member access")
	}
}

func TestCompileSupportsArrowFunctionParameterDestructuring(t *testing.T) {
	source := `
function main(args) {
  var read = ({ value }) => value;
  return read({ value: 3 });
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@__jayess_lambda_0") {
		t.Fatalf("expected arrow function with destructured parameters to lower successfully")
	}
}

func TestCompileSupportsDestructuringRestElementsAndDefaults(t *testing.T) {
	source := `
function main(args) {
  const { a = 1, ...rest } = { b: 2 };
  const [x = 3, ...tail] = [undefined, 4, 5];
  print(rest.b);
  print(tail.length);
  return a + x;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_object_rest") || !strings.Contains(irText, "@jayess_value_array_slice") {
		t.Fatalf("expected destructuring rest elements to lower through object-rest and array-slice helpers")
	}
	if !strings.Contains(irText, "undefined") {
		t.Fatalf("expected destructuring defaults to keep undefined checks in lowered IR")
	}
}

func TestCompileSupportsNestedDestructuringDefaults(t *testing.T) {
	source := `
function main(args) {
  const { point: { x = 4 } } = { point: {} };
  const [first = 2] = [];
  return x + first;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_get_member") {
		t.Fatalf("expected nested destructuring defaults to compile through member access")
	}
}

func TestCompileSupportsArithmeticAssignmentOperators(t *testing.T) {
	source := `
function main(args) {
  var total = 2;
  total += 3;
  total *= 4;
  total -= 2;
  total /= 2;
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_to_number") || !strings.Contains(irText, "store ptr") {
		t.Fatalf("expected arithmetic assignment operators to lower through dynamic numeric coercion and reassignment")
	}
}

func TestCompileSupportsNullishAndLogicalAssignmentOperators(t *testing.T) {
	source := `
function main(args) {
  var maybe = undefined;
  maybe ??= "kimchi";
  var name = "";
  name ||= "jjigae";
  var ready = true;
  ready &&= false;
  print(maybe, name, ready);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_is_nullish") || !strings.Contains(irText, "@jayess_value_is_truthy") {
		t.Fatalf("expected nullish/logical assignment operators to lower through runtime nullish/truthy checks")
	}
}

func TestCompileSupportsTemplateInterpolation(t *testing.T) {
	source := "function main(args) {\n  var something = \"kimchi\";\n  print(`${something} fefqeq`);\n  return 0;\n}\n"

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_template_string") {
		t.Fatalf("expected template interpolation to lower through template helper")
	}
}

func TestCompileRejectsCapturingFunctionExpressions(t *testing.T) {
	source := `
function main(args) {
  var base = 2;
  var add = (value) => value + base;
  return add(3);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@__jayess_lambda_0") {
		t.Fatalf("expected lowered closure helper in LLVM IR")
	}
	if !strings.Contains(irText, "base") {
		t.Fatalf("expected captured environment access in LLVM IR")
	}
}

func TestCompileSupportsNestedClosureCapture(t *testing.T) {
	source := `
function main(args) {
  var outer = 10;
  var make = (value) => {
    var read = () => outer + value;
    return read();
  };
  return make(2);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@__jayess_lambda_0") || !strings.Contains(irText, "@__jayess_lambda_1") {
		t.Fatalf("expected nested lowered closure helpers in LLVM IR")
	}
}

func TestCompileSupportsArrowCapturingThisInsideClassMethod(t *testing.T) {
	source := `
class Counter {
  value = 3;

  read() {
    var get = () => this.value;
    return get();
  }
}

function main(args) {
  var counter = new Counter();
  return counter.read();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@Counter__read") {
		t.Fatalf("expected class method to compile with arrow closure")
	}
}

func TestCompileSupportsArrowCapturingSuperInsideClassMethod(t *testing.T) {
	source := `
class Base {
  read() {
    return 2;
  }
}

class Child extends Base {
  readTwice() {
    var callBase = () => super.read();
    return callBase() + callBase();
  }
}

function main(args) {
  var child = new Child();
  return child.readTwice();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@Base__read") {
		t.Fatalf("expected captured super call to lower to base method symbol")
	}
}

func TestCompileSupportsFunctionExpressionCapturingSuperProperty(t *testing.T) {
	source := `
class Base {
  value = 4;
}

class Child extends Base {
  total() {
    var getBase = function() {
      return super.value;
    };
    return getBase() + 1;
  }
}

function main(args) {
  var child = new Child();
  return child.total();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@Child__total") {
		t.Fatalf("expected derived class method with captured super property to compile")
	}
}

func TestCompileSupportsFirstClassNamedFunctionValues(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  const fn = greet;
  print(fn);
  return fn();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_from_function") {
		t.Fatalf("expected function values to be boxed in LLVM IR")
	}
	if !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected function value invocation helpers in LLVM IR")
	}
}

func TestCompileSupportsFirstClassClosureInvocation(t *testing.T) {
	source := `
function main(args) {
  const add = (value) => value + 2;
  return add(3);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_from_function") {
		t.Fatalf("expected closure values to lower through function boxing")
	}
}

func TestCompileSupportsObjectStoredFunctionInvocation(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  var obj = { cb: greet };
  return obj.cb();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_object_set_value") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected object-stored function invocation to use object storage and first-class invoke helpers")
	}
}

func TestCompileSupportsArrayStoredFunctionInvocation(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  var items = [greet];
  return items[0]();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_array_set_value") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected array-stored function invocation to use array storage and first-class invoke helpers")
	}
}

func TestCompileSupportsObjectStoredExtractedMethodInvocation(t *testing.T) {
	source := `
class Dog {
  sound() {
    return "dog";
  }
}

function main(args) {
  var dog = new Dog();
  var obj = { speak: dog.sound };
  return obj.speak();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@__jayess_dispatch__sound__0") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected extracted method stored in object to invoke through dispatch and first-class function helpers")
	}
}

func TestCompileSupportsReturnedFunctionImmediateInvocation(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function factory() {
  return greet;
}

function main(args) {
  return factory()();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_from_function") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected returned function invocation to use first-class function helpers")
	}
}

func TestCompileSupportsCallbackParameterInvocation(t *testing.T) {
	source := `
function run(fn) {
  return fn();
}

function greet() {
  return "hello";
}

function main(args) {
  return run(greet);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_function_ptr") {
		t.Fatalf("expected callback invocation to use first-class invoke helpers")
	}
}

func TestCompileSupportsFunctionProperties(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  greet.label = "wave";
  print(greet.label);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_set_member") || !strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected function property access to use dynamic member helpers")
	}
}

func TestCompileSupportsFunctionCallMethod(t *testing.T) {
	source := `
function greet(name) {
  print(name);
  return 0;
}

function main(args) {
  return greet.call(null, "kimchi");
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_function_ptr") {
		t.Fatalf("expected function.call to use first-class invoke helpers")
	}
}

func TestCompileSupportsDynamicThisViaCall(t *testing.T) {
	source := `
function readName() {
  print(this.name);
  return 0;
}

function main(args) {
  return readName.call({ name: "kimchi" });
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_current_this") || !strings.Contains(irText, "@jayess_push_this") {
		t.Fatalf("expected ordinary function this to flow through runtime this helpers")
	}
}

func TestCompileSupportsFunctionApplyMethod(t *testing.T) {
	source := `
function greet(name) {
  print(name);
  return 0;
}

function main(args) {
  return greet.apply(null, ["kimchi"]);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_array_length") || !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected function.apply to use bounded apply helpers")
	}
}

func TestCompileSupportsFunctionBindMethod(t *testing.T) {
	source := `
function greet() {
  return "hello";
}

function main(args) {
  const bound = greet.bind(null);
  return bound();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_function_ptr") {
		t.Fatalf("expected function.bind result to remain a first-class callable value")
	}
}

func TestCompileSupportsBoundThisViaBind(t *testing.T) {
	source := `
function readName() {
  print(this.name);
  return 0;
}

function main(args) {
  const bound = readName.bind({ name: "kimchi" });
  return bound();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_bind") || !strings.Contains(irText, "@jayess_current_this") {
		t.Fatalf("expected bind to preserve thisArg for ordinary functions")
	}
}

func TestCompileSupportsTypeof(t *testing.T) {
	source := `
function main(args) {
  var value = "kimchi";
  print(typeof value);
  print(typeof 1);
  print(typeof {});
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_typeof") {
		t.Fatalf("expected typeof on dynamic values to use runtime typeof helper")
	}
}

func TestCompileSupportsInstanceofWithInheritance(t *testing.T) {
	source := `
class Animal {}

class Dog extends Animal {}

function main(args) {
  var dog = new Dog();
  if (dog instanceof Animal) {
    return 1;
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "__jayess_is_Animal") || !strings.Contains(irText, "@jayess_value_instanceof") {
		t.Fatalf("expected instanceof to use runtime ancestry markers")
	}
}

func TestCompileSupportsDynamicInstanceofFallback(t *testing.T) {
	source := `
class Dog {}

function main(args) {
  var dog = new Dog();
  var Type = Dog;
  if (dog instanceof Type) {
    return 1;
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_instanceof") {
		t.Fatalf("expected dynamic instanceof to use runtime fallback helper")
	}
}

func TestCompileSupportsFunctionBindWithPreBoundArguments(t *testing.T) {
	source := `
function greet(name) {
  print(name);
  return 0;
}

function main(args) {
  const bound = greet.bind(null, "kimchi");
  return bound();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_bind") || !strings.Contains(irText, "@jayess_value_merge_bound_args") {
		t.Fatalf("expected pre-bound function arguments to use bind and merged apply helpers")
	}
}

func TestCompileSupportsPartialBindOnDynamicFunctionValues(t *testing.T) {
	source := `
function add(a, b) {
  return a + b;
}

function main(args) {
  const inc = add.bind(null, 1);
  return inc(2);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_bind") || !strings.Contains(irText, "@jayess_value_merge_bound_args") {
		t.Fatalf("expected partial bind to lower through runtime bind helpers")
	}
}

func TestCompileSupportsNewTargetInConstructors(t *testing.T) {
	source := `
class Person {
  constructor() {
    this.kind = typeof new.target;
  }
}

function main(args) {
  var person = new Person();
  if (person.kind == "function") {
    return 1;
  }
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_from_function") {
		t.Fatalf("expected new.target to lower through boxed constructor function values")
	}
}

func TestCompileSupportsArrayLengthProperty(t *testing.T) {
	source := `
function main(args) {
  var values = [1, 2, 3];
  return values.length;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_array_length") {
		t.Fatalf("expected array length to use runtime length helpers")
	}
}

func TestCompileSupportsArrayPushMethod(t *testing.T) {
	source := `
function main(args) {
  var values = [];
  return values.push(1);
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_array_push") {
		t.Fatalf("expected array push to lower through runtime push helpers")
	}
}

func TestCompileSupportsArrayPopMethod(t *testing.T) {
	source := `
function main(args) {
  var values = [1];
  return values.pop();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_array_pop") {
		t.Fatalf("expected array pop to lower through runtime pop helpers")
	}
}

func TestCompileSupportsArrayShiftUnshiftSliceAndForEach(t *testing.T) {
	source := `
function addOne(value) {
  print(value);
  return 0;
}

function main(args) {
  var values = [1, 2];
  values.unshift(0);
  values.forEach(addOne);
  print(values.slice(1).length);
  return values.shift();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_array_unshift") || !strings.Contains(irText, "@jayess_value_array_shift") || !strings.Contains(irText, "@jayess_value_array_slice") {
		t.Fatalf("expected shift/unshift/slice to lower through runtime array helpers")
	}
	if !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected forEach callback invocation to use function-value invoke helpers")
	}
}

func TestCompileSupportsForOfAndForIn(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  for (var value of [1, 2, 3]) {
    total = total + value;
  }
  var obj = { a: 1, b: 2 };
  for (const key in obj) {
    print(key);
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_object_keys") || !strings.Contains(irText, "@jayess_value_array_length") {
		t.Fatalf("expected for...of/in lowering to use array length and object key helpers")
	}
}

func TestCompileSupportsSwitch(t *testing.T) {
	source := `
function main(args) {
  var value = 2;
  switch (value) {
    case 1:
      return 1;
    case 2:
      return 2;
    default:
      return 0;
  }
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "if.then") {
		t.Fatalf("expected switch to lower through conditional branches")
	}
}

func TestCompileSupportsComments(t *testing.T) {
	source := `
// leading comment
function main(args) {
  /* block comment */
  return 1;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.LLVMIR) == 0 {
		t.Fatalf("expected LLVM IR output")
	}
}

func TestCompileSupportsTemplateStrings(t *testing.T) {
	source := `
function main(args) {
  var name = "kimchi";
  print(` + "`hello ${name}`" + `);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_template_string") {
		t.Fatalf("expected template strings to lower through runtime template helper")
	}
}

func TestCompileSupportsRuntimeCompileBuiltin(t *testing.T) {
	source := `
function main(args) {
  var result = compile("function main() { return 0; }", "build/runtime-compiled");
  console.log(result.ok, result.output, result.stdout, result.stderr, result.error);
  var configured = compile("function main() { return 0; }", { output: "build/runtime-compiled-configured", emit: "exe", warnings: "default" });
  console.log(configured.ok, configured.output, configured.status);
  var invalid = compile("function main() { return 0; }", { emit: "bad" });
  console.log(invalid.ok, invalid.error);
  var fileResult = compileFile("src/main.js", { output: "build/runtime-compiled-file", emit: "exe" });
  console.log(fileResult.ok, fileResult.stderr);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_compile") {
		t.Fatalf("expected compile builtin to lower through runtime compile helper")
	}
	if !strings.Contains(irText, "@jayess_std_compile_file") {
		t.Fatalf("expected compileFile builtin to lower through runtime compile file helper")
	}
}

func TestCompileSupportsObjectLiteralMethodsAndComputedKeys(t *testing.T) {
	source := `
function main(args) {
  var key = "name";
  var obj = {
    [key]: "kimchi",
    greet() {
      return "hello";
    }
  };
  print(obj[key]);
  print(obj.greet());
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_set_computed_member") || !strings.Contains(irText, "@jayess_value_from_function") {
		t.Fatalf("expected computed keys and object literal methods to lower through runtime object/function helpers")
	}
}

func TestLowerNestedClosureRewritesCapturedEnvValues(t *testing.T) {
	source := `
function main(args) {
  var outer = 10;
  var make = (value) => {
    var read = () => outer + value;
    return read();
  };
  return make(2);
}
`

	p := parser.New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	program, err = lowerFunctionExpressions(program)
	if err != nil {
		t.Fatalf("lowerFunctionExpressions returned error: %v", err)
	}
	program, err = lowerClasses(program)
	if err != nil {
		t.Fatalf("lowerClasses returned error: %v", err)
	}

	for _, fn := range program.Functions {
		if strings.HasPrefix(fn.Name, "__jayess_lambda_") && functionBodyContainsIdentifier(fn.Body, "outer") {
			t.Fatalf("expected lowered closure body to avoid raw outer identifier in %s: %v", fn.Name, collectIdentifiers(fn.Body))
		}
	}
}

func collectIdentifiers(statements []ast.Statement) []string {
	var out []string
	for _, stmt := range statements {
		collectIdentifiersFromStatement(stmt, &out)
	}
	return out
}

func collectIdentifiersFromStatement(stmt ast.Statement, out *[]string) {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		collectIdentifiersFromExpression(stmt.Value, out)
	case *ast.AssignmentStatement:
		collectIdentifiersFromExpression(stmt.Target, out)
		collectIdentifiersFromExpression(stmt.Value, out)
	case *ast.ReturnStatement:
		collectIdentifiersFromExpression(stmt.Value, out)
	case *ast.ExpressionStatement:
		collectIdentifiersFromExpression(stmt.Expression, out)
	case *ast.DeleteStatement:
		collectIdentifiersFromExpression(stmt.Target, out)
	case *ast.IfStatement:
		collectIdentifiersFromExpression(stmt.Condition, out)
		for _, child := range stmt.Consequence {
			collectIdentifiersFromStatement(child, out)
		}
		for _, child := range stmt.Alternative {
			collectIdentifiersFromStatement(child, out)
		}
	case *ast.WhileStatement:
		collectIdentifiersFromExpression(stmt.Condition, out)
		for _, child := range stmt.Body {
			collectIdentifiersFromStatement(child, out)
		}
	case *ast.ForStatement:
		if stmt.Init != nil {
			collectIdentifiersFromStatement(stmt.Init, out)
		}
		if stmt.Condition != nil {
			collectIdentifiersFromExpression(stmt.Condition, out)
		}
		if stmt.Update != nil {
			collectIdentifiersFromStatement(stmt.Update, out)
		}
		for _, child := range stmt.Body {
			collectIdentifiersFromStatement(child, out)
		}
	}
}

func collectIdentifiersFromExpression(expr ast.Expression, out *[]string) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		*out = append(*out, expr.Name)
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			collectIdentifiersFromExpression(property.Value, out)
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			collectIdentifiersFromExpression(element, out)
		}
	case *ast.ClosureExpression:
		collectIdentifiersFromExpression(expr.Environment, out)
	case *ast.BinaryExpression:
		collectIdentifiersFromExpression(expr.Left, out)
		collectIdentifiersFromExpression(expr.Right, out)
	case *ast.ComparisonExpression:
		collectIdentifiersFromExpression(expr.Left, out)
		collectIdentifiersFromExpression(expr.Right, out)
	case *ast.LogicalExpression:
		collectIdentifiersFromExpression(expr.Left, out)
		collectIdentifiersFromExpression(expr.Right, out)
	case *ast.UnaryExpression:
		collectIdentifiersFromExpression(expr.Right, out)
	case *ast.IndexExpression:
		collectIdentifiersFromExpression(expr.Target, out)
		collectIdentifiersFromExpression(expr.Index, out)
	case *ast.MemberExpression:
		collectIdentifiersFromExpression(expr.Target, out)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			collectIdentifiersFromExpression(arg, out)
		}
	case *ast.InvokeExpression:
		collectIdentifiersFromExpression(expr.Callee, out)
		for _, arg := range expr.Arguments {
			collectIdentifiersFromExpression(arg, out)
		}
	case *ast.NewExpression:
		collectIdentifiersFromExpression(expr.Callee, out)
		for _, arg := range expr.Arguments {
			collectIdentifiersFromExpression(arg, out)
		}
	}
}

func functionBodyContainsIdentifier(statements []ast.Statement, name string) bool {
	for _, stmt := range statements {
		if statementContainsIdentifier(stmt, name) {
			return true
		}
	}
	return false
}

func statementContainsIdentifier(stmt ast.Statement, name string) bool {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		return expressionContainsIdentifier(stmt.Value, name)
	case *ast.AssignmentStatement:
		return expressionContainsIdentifier(stmt.Target, name) || expressionContainsIdentifier(stmt.Value, name)
	case *ast.ReturnStatement:
		return expressionContainsIdentifier(stmt.Value, name)
	case *ast.ExpressionStatement:
		return expressionContainsIdentifier(stmt.Expression, name)
	case *ast.DeleteStatement:
		return expressionContainsIdentifier(stmt.Target, name)
	case *ast.IfStatement:
		return expressionContainsIdentifier(stmt.Condition, name) || functionBodyContainsIdentifier(stmt.Consequence, name) || functionBodyContainsIdentifier(stmt.Alternative, name)
	case *ast.WhileStatement:
		return expressionContainsIdentifier(stmt.Condition, name) || functionBodyContainsIdentifier(stmt.Body, name)
	case *ast.ForStatement:
		return (stmt.Init != nil && statementContainsIdentifier(stmt.Init, name)) ||
			(stmt.Condition != nil && expressionContainsIdentifier(stmt.Condition, name)) ||
			(stmt.Update != nil && statementContainsIdentifier(stmt.Update, name)) ||
			functionBodyContainsIdentifier(stmt.Body, name)
	default:
		return false
	}
}

func expressionContainsIdentifier(expr ast.Expression, name string) bool {
	switch expr := expr.(type) {
	case *ast.Identifier:
		return expr.Name == name
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			if expressionContainsIdentifier(property.Value, name) {
				return true
			}
		}
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			if expressionContainsIdentifier(element, name) {
				return true
			}
		}
	case *ast.ClosureExpression:
		return expressionContainsIdentifier(expr.Environment, name)
	case *ast.BinaryExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.ComparisonExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.LogicalExpression:
		return expressionContainsIdentifier(expr.Left, name) || expressionContainsIdentifier(expr.Right, name)
	case *ast.UnaryExpression:
		return expressionContainsIdentifier(expr.Right, name)
	case *ast.IndexExpression:
		return expressionContainsIdentifier(expr.Target, name) || expressionContainsIdentifier(expr.Index, name)
	case *ast.MemberExpression:
		return expressionContainsIdentifier(expr.Target, name)
	case *ast.CallExpression:
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
	case *ast.InvokeExpression:
		if expressionContainsIdentifier(expr.Callee, name) {
			return true
		}
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
	case *ast.NewExpression:
		if expressionContainsIdentifier(expr.Callee, name) {
			return true
		}
		for _, arg := range expr.Arguments {
			if expressionContainsIdentifier(arg, name) {
				return true
			}
		}
	}
	return false
}
