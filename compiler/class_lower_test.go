package compiler

import (
	"errors"
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
	if !strings.Contains(irText, "@jayess_fn___jayess_dispatch__sound__0") {
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
	if !strings.Contains(irText, "@jayess_fn___jayess_dispatch__sound__0") {
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
	if !strings.Contains(irText, "@jayess_fn_Dog__kind") {
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

func TestCompileSupportsWeakMapWeakSetStandardLibrarySurface(t *testing.T) {
	source := `
function main(args) {
  var weakMap = new WeakMap();
  var weakSet = new WeakSet();
  var key = {};
  weakMap.set(key, "kimchi");
  print(weakMap.get(key));
  print(weakMap.has(key));
  print(weakMap.delete(key));
  weakSet.add(key);
  print(weakSet.has(key));
  print(weakSet.delete(key));
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_std_weak_map_new",
		"@jayess_std_weak_set_new",
		"@jayess_value_get_member",
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

func TestCompileSupportsPropertyTypeAnnotationsAndLocalInference(t *testing.T) {
	source := `
class Box {
  value: number = 1;
  payload: object = { label: "kimchi" };
  items: array = [1, 2, 3];
  ready: boolean = true;
  note: any = "broth";
}

function add(left: number, right: number): number {
  return left + right;
}

function main(args) {
  const inferred = 2;
  const text = "ramen";
  const flag = true;
  const huge: bigint = 1n;
  var box = new Box();
  console.log(add(inferred, 3), box.value, text, flag, huge, box.payload.label, box.items.length, box.note);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesClassFieldTypeAnnotations(t *testing.T) {
	source := `
class Box {
  value: number = "kimchi";
}

function main(args) {
  return 0;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if !strings.Contains(err.Error(), "cannot initialize number field value with string") {
		t.Fatalf("expected class field type annotation error, got %v", err)
	}
}

func TestCompileSupportsUnknownNeverNullAndUndefinedTypeAnnotations(t *testing.T) {
	source := `
function fail(message: string): never {
  throw new Error(message);
}

function main(args) {
  var mystery: unknown = 1;
  mystery = "kimchi";
  var nothing: null = null;
  var missing: undefined = undefined;
  if (false) {
    fail("boom");
  }
  console.log("typed");
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesUnknownNeverNullAndUndefinedTypeRules(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "null initializer mismatch",
			source: `
function main(args) {
  var value: null = undefined;
  return 0;
}
`,
			message: "cannot initialize null variable value with undefined",
		},
		{
			name: "undefined initializer mismatch",
			source: `
function main(args) {
  var value: undefined = null;
  return 0;
}
`,
			message: "cannot initialize undefined variable value with null",
		},
		{
			name: "unknown cannot flow into number",
			source: `
function main(args) {
  var mystery: unknown = 1;
  var count: number = mystery;
  return 0;
}
`,
			message: "cannot initialize number variable count with unknown",
		},
		{
			name: "never requires terminal throw",
			source: `
function fail(): never {
  return undefined;
}

function main(args) {
  fail();
  return 0;
}
`,
			message: "function fail must return never, got undefined",
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

func TestCompileSupportsTypeAliasesAndAsAssertions(t *testing.T) {
	source := `
type Count = number;
type Label = string;
type Total = Count;

function main(args) {
  var raw = 3;
  var count: Total = raw as Count;
  var title = "kimchi";
  var label: Label = title as Label;
  console.log(count + 1, label);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileRejectsInvalidTypeAliasesAndAsAssertions(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "duplicate alias",
			source: `
type Count = number;
type Count = string;

function main(args) {
  return 0;
}
`,
			message: "duplicate type alias Count",
		},
		{
			name: "alias cycle",
			source: `
type Left = Right;
type Right = Left;

function main(args) {
  return 0;
}
`,
			message: "type alias cycle detected involving",
		},
		{
			name: "unknown alias target",
			source: `
type Count = Missing;

function main(args) {
  return 0;
}
`,
			message: "unknown type alias Missing",
		},
		{
			name: "cast to never",
			source: `
function main(args) {
  var value = 1 as never;
  return 0;
}
`,
			message: "casts to never are not supported",
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

func TestCompileSupportsStructuredTypeAnnotations(t *testing.T) {
	source := `
interface User {
  readonly id: number;
  name: string;
  age?: number;
}

type Pair = [number, string];
type Mapper = (number, string) => boolean;

function makeMapper(): Mapper {
  return (count: number, label: string): boolean => count > 0;
}

function main(args) {
  var user: User = { id: 1, name: "kimchi" };
  var pair: Pair = [3, "ramen"];
  var mapper: Mapper = makeMapper();
  var directMapper: (number, string) => boolean = mapper;
  console.log(user.name);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesStructuredTypeAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "tuple length mismatch",
			source: `
type Pair = [number, string];

function main(args) {
  var pair: Pair = [1];
  return 0;
}
`,
			message: "cannot initialize [number,string] variable pair with array",
		},
		{
			name: "missing required interface property",
			source: `
interface User {
  id: number;
  name: string;
  age?: number;
}

function main(args) {
  var user: User = { id: 1 };
  return 0;
}
`,
			message: "cannot initialize {id:number,name:string,age?:number} variable user with object",
		},
		{
			name: "callable mismatch",
			source: `
type Mapper = (number, string) => boolean;

function main(args) {
  var mapper: Mapper = 1;
  return 0;
}
`,
			message: "cannot initialize (number,string)=>boolean variable mapper with number",
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

func TestCompileSupportsReadonlyAndIndexSignatureTypes(t *testing.T) {
	source := `
interface Dictionary {
  [key: string]: number;
}

function main(args) {
  var values: Dictionary = { kimchi: 3, ramen: 5 };
  values.extra = 8;
  values["bonus"] = 13;
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesReadonlyAndIndexSignatureTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "readonly property assignment",
			source: `
interface User {
  readonly id: number;
  name: string;
}

function main(args) {
  var user: User = { id: 1, name: "kimchi" };
  user.id = 2;
  return 0;
}
`,
			message: "cannot assign to readonly property id",
		},
		{
			name: "readonly index signature assignment",
			source: `
interface Dictionary {
  readonly [key: string]: number;
}

function main(args) {
  var values: Dictionary = { kimchi: 3 };
  values.extra = 8;
  return 0;
}
`,
			message: "cannot assign to readonly property extra",
		},
		{
			name: "index signature value mismatch",
			source: `
interface Dictionary {
  [key: string]: number;
}

function main(args) {
  var values: Dictionary = { kimchi: 3 };
  values.extra = "bad";
  return 0;
}
`,
			message: "cannot assign string to number",
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

func TestCompileSupportsLiteralAndUnionTypes(t *testing.T) {
	source := `
type Status = "ok" | "error";
type Count = 1 | 2;
type Tagged = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  var status: Status = "ok";
  var count: Count = 1;
  var tagged: Tagged = { kind: "ok", value: 3 };
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesLiteralAndUnionTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "string literal mismatch",
			source: `
type Status = "ok";

function main(args) {
  var status: Status = "error";
  return 0;
}
`,
			message: "cannot initialize \"ok\" variable status with string",
		},
		{
			name: "string union mismatch",
			source: `
type Status = "ok" | "error";

function main(args) {
  var status: Status = "pending";
  return 0;
}
`,
			message: "cannot initialize \"ok\"|\"error\" variable status with string",
		},
		{
			name: "tagged union mismatch",
			source: `
type Tagged = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  var tagged: Tagged = { kind: "ok", message: "bad" };
  return 0;
}
`,
			message: "cannot initialize {kind:\"ok\",value:number}|{kind:\"error\",message:string} variable tagged with object",
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

func TestCompileSupportsEnums(t *testing.T) {
	source := `
enum Status {
  Ok,
  Error = 3,
  Ready = "ready",
}

function main(args) {
  var numeric = Status.Ok;
  var named: Status = "ready";
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesEnumTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "enum value mismatch",
			source: `
enum Status {
  Ok,
  Error = 3,
}

function main(args) {
  var status: Status = "bad";
  return 0;
}
`,
			message: "cannot initialize 0|3 variable status with string",
		},
		{
			name: "enum implicit after string member",
			source: `
enum Status {
  Ready = "ready",
  Done,
}

function main(args) {
  return 0;
}
`,
			message: "enum member Done requires an explicit initializer after a non-numeric member",
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

func TestCompileSupportsIntersectionTypes(t *testing.T) {
	source := `
type Combined = { id: number } & { name: string };

function main(args) {
  var value: Combined = { id: 1, name: "kimchi" };
  console.log(value.id, value.name);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesIntersectionTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "missing intersected property",
			source: `
type Combined = { id: number } & { name: string };

function main(args) {
  var value: Combined = { id: 1 };
  return 0;
}
`,
			message: "cannot initialize {id:number}&{name:string} variable value with object",
		},
		{
			name: "readonly in intersection",
			source: `
type Combined = { readonly id: number } & { name: string };

function main(args) {
  var value: Combined = { id: 1, name: "kimchi" };
  value.id = 2;
  return 0;
}
`,
			message: "cannot assign to readonly property id",
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

func TestCompileSupportsDiscriminatedUnionNarrowing(t *testing.T) {
	source := `
type Result = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  var result: Result = { kind: "ok", value: 3 };
  if (result.kind === "ok") {
    console.log(result.value);
  } else {
    console.log(result.message);
  }
  switch (result.kind) {
    case "ok": {
      console.log(result.value);
      break;
    }
    case "error": {
      console.log(result.message);
      break;
    }
  }
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileSupportsRuntimeTypeChecks(t *testing.T) {
	source := `
type Result = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  var value: string | number = "kimchi";
  if (value is string) {
    var text: string = value;
    console.log(text);
  }

  var pair = [1, "ok"];
  var ok = pair is [number, string];
  var result = { kind: "ok", value: 3 } is Result;
  console.log(ok, result);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(result.LLVMIR) == 0 {
		t.Fatalf("expected runtime type check program to emit LLVM IR")
	}
}

func TestCompileUsesRuntimeManagedExceptionsWithoutLLVMEH(t *testing.T) {
	result, err := Compile(`
function main(args) {
  try {
    throw "boom";
  } catch (err) {
    console.log(err);
  }
  return 0;
}
`, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, fragment := range []string{
		"@jayess_throw(",
		"@jayess_has_exception()",
		"@jayess_take_exception()",
		"@jayess_push_call_frame(",
		"@jayess_pop_call_frame()",
	} {
		if !strings.Contains(irText, fragment) {
			t.Fatalf("expected runtime-managed exception fragment %q, got:\n%s", fragment, irText)
		}
	}
	for _, forbidden := range []string{" invoke ", "\ninvoke ", "landingpad", "personality "} {
		if strings.Contains(irText, forbidden) {
			t.Fatalf("expected no LLVM EH construct %q, got:\n%s", forbidden, irText)
		}
	}
}

func TestCompileSupportsGenericAliasesAndConstraints(t *testing.T) {
	source := `
type Box<T> = { value: T };
type Named<T extends string | number> = { id: T, name: string };
interface Pair<T> {
  left: T,
  right: T,
}

function main(args) {
  var box: Box<number> = { value: 1 };
  var named: Named<string> = { id: "kimchi", name: "ok" };
  var pair: Pair<boolean> = { left: true, right: false };
  console.log(box.value, named.id, pair.left);
  return 0;
}
`

	if _, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
}

func TestCompileEnforcesGenericAliasConstraints(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		message string
	}{
		{
			name: "missing type arguments",
			source: `
type Box<T> = { value: T };

function main(args) {
  var box: Box = { value: 1 };
  return 0;
}
`,
			message: "generic type alias Box requires type arguments",
		},
		{
			name: "constraint violation",
			source: `
type Named<T extends string | number> = { id: T, name: string };

function main(args) {
  var named: Named<boolean> = { id: true, name: "ok" };
  return 0;
}
`,
			message: "type argument boolean does not satisfy constraint string|number for T",
		},
		{
			name: "wrong arity",
			source: `
type Pair<T, U> = { left: T, right: U };

function main(args) {
  var pair: Pair<number> = { left: 1, right: 2 };
  return 0;
}
`,
			message: "type alias Pair expects 2 type arguments",
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

func TestCompileReportsLifetimeEscapeWarnings(t *testing.T) {
	source := `
var sink = null;

function buildCounter() {
  var count = 1;
  var box = { value: count };
  sink = box;
  return () => count;
}

function main(args) {
  buildCounter();
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	var lifetimeWarnings []Diagnostic
	for _, warning := range result.Warnings {
		if warning.Category == "lifetime" {
			lifetimeWarnings = append(lifetimeWarnings, warning)
		}
	}
	if len(lifetimeWarnings) < 3 {
		t.Fatalf("expected at least 3 lifetime warnings, got %#v", lifetimeWarnings)
	}
	expected := []struct {
		line    int
		column  int
		message string
	}{
		{line: 7, column: 10, message: "local box escapes via assignment to global state"},
		{line: 8, column: 16, message: "local count escapes via closure capture"},
		{line: 8, column: 16, message: "local count escapes via return"},
	}
	for _, want := range expected {
		found := false
		for _, warning := range lifetimeWarnings {
			if warning.Code == "JY400" && warning.Line == want.line && warning.Column == want.column && warning.Message == want.message {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected lifetime warning %#v in %#v", want, lifetimeWarnings)
		}
	}
}

func TestCompileLifetimeWarningPolicyCanEscalate(t *testing.T) {
	source := `
function makeBox() {
  var count = 1;
  return { value: count };
}

function main(args) {
  makeBox();
  return 0;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc", WarningPolicy: "error"})
	if err == nil {
		t.Fatalf("expected warning policy to escalate lifetime warning")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Category != "lifetime" || compileErr.Diagnostic.Code != "JY400" {
		t.Fatalf("expected lifetime diagnostic, got %#v", compileErr.Diagnostic)
	}
	if !strings.Contains(compileErr.Diagnostic.Message, "local count escapes via return") {
		t.Fatalf("expected lifetime return message, got %#v", compileErr.Diagnostic)
	}
}

func TestCompileReportsConservativeEscapeWarnings(t *testing.T) {
	source := `
extern function retain(value);

function main(args) {
  var count = 1;
  var box = {};
  var items = [];
  var update = () => {
    count = count + 1;
    return count;
  };
  box.value = count;
  items[0] = count;
  retain(count);
  return update();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	var lifetimeWarnings []Diagnostic
	for _, warning := range result.Warnings {
		if warning.Category == "lifetime" {
			lifetimeWarnings = append(lifetimeWarnings, warning)
		}
	}

	wants := []string{
		"local count escapes via closure capture",
		"local count escapes via assignment to outer scope",
		"local count escapes via return",
		"local count escapes via object storage",
		"local count escapes via array storage",
		"local count escapes via call to unknown or external function",
	}
	for _, want := range wants {
		found := false
		for _, warning := range lifetimeWarnings {
			if warning.Message == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected conservative lifetime warning %q in %#v", want, lifetimeWarnings)
		}
	}
}

func TestCompileExposesEligibleNonEscapingLocals(t *testing.T) {
	source := `
function helper() {
  var count = 1;
  var box = { total: count + 1 };
  print(box.total);
  return 0;
}

function main(args) {
  helper();
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	found := map[string]bool{}
	for _, item := range result.LifetimeReport.Eligible {
		if item.Function == "helper" {
			found[item.Name] = true
		}
	}
	for _, want := range []string{"count", "box"} {
		if !found[want] {
			t.Fatalf("expected eligible local %q in %#v", want, result.LifetimeReport.Eligible)
		}
	}
}

func TestCompileExposesEligibleNonEscapingParameters(t *testing.T) {
	workdir := t.TempDir()
	nativeDir := filepath.Join(workdir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "borrow.c"), []byte(`int jayess_test_borrow(void) { return 0; }`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "borrow.bind.js"), []byte(`const f = () => {};
export const borrow = f;
export const retain = f;
export default {
  sources: ["./borrow.c"],
  exports: {
    borrow: { symbol: "jayess_test_borrow", type: "function", borrowsArgs: true },
    retain: { symbol: "jayess_test_retain", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { borrow, retain } from "./native/borrow.bind.js";

function keep(label) {
  return borrow(label);
}

function escape(label) {
  return retain(label);
}

function main(args) {
  keep("ok");
  escape("nope");
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	foundKeep := false
	foundEscape := false
	for _, item := range result.LifetimeReport.EligibleParams {
		if item.Function == "keep" && item.Name == "label" {
			foundKeep = true
		}
		if item.Function == "escape" && item.Name == "label" {
			foundEscape = true
		}
	}
	if !foundKeep {
		t.Fatalf("expected eligible parameter keep(label) in %#v", result.LifetimeReport.EligibleParams)
	}
	if foundEscape {
		t.Fatalf("expected escape(label) to be excluded, got %#v", result.LifetimeReport.EligibleParams)
	}
}

func TestCompileDoesNotExposeSyntheticEligibleLocals(t *testing.T) {
	source := `
function buildClosures() {
  var items = [];
  for (var i = 0; i < 6; i = i + 1) {
    var value = i;
    if (value == 1) {
      continue;
    }
    items.push(() => value);
    if (value == 4) {
      break;
    }
  }
  return items;
}

function main(args) {
  var items = buildClosures();
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	for _, item := range result.LifetimeReport.Eligible {
		if strings.HasPrefix(item.Name, "__jayess_") {
			t.Fatalf("unexpected synthetic eligible local %#v", item)
		}
	}
}

func TestCompileEmitsScopeExitCleanupForEligibleDynamicLocals(t *testing.T) {
	source := `
function makeValue() {
  return undefined;
}

function freshNumber() {
  return 11;
}

function inner() {
  var value = makeValue();
  return 1;
}

function main(args) {
  freshNumber();
  {
    const scoped = makeValue();
  }
  return inner();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	irText := string(result.LLVMIR)
	if count := strings.Count(irText, "call void @jayess_value_free_unshared("); count < 3 {
		t.Fatalf("expected at least three cleanup calls in LLVM IR, got %d:\n%s", count, irText)
	}
	if !strings.Contains(irText, "call ptr @jayess_fn_freshNumber()") {
		t.Fatalf("expected direct call to freshNumber in LLVM IR:\n%s", irText)
	}
}

func TestCompileEmitsDiscardedFreshExpressionCleanup(t *testing.T) {
	source := `
class FreshBox {}
class PlainCtorBox {
  constructor() {
    this.kind = "plain";
  }
}

class FreshReturnCtorBox {
  constructor() {
    return { kind: "alt-fresh" };
  }
}

function freshObject() {
  return { kind: "fresh-call" };
}

function freshInvokeObject() {
  return { kind: "invoke-fresh" };
}

function freshBox() {
  return new FreshBox();
}

function freshSwitchCase() {
  switch ("case-" + "a") {
    case "case-a":
      break;
    case "case-b":
      break;
  }
}

function boundOffset(offset, x) {
  return x + offset;
}

function largeOffset(x) {
  return x + 20;
}

function boundGreaterThan(min, x) {
  return x > min;
}

function boundEquals(expected, x) {
  return x == expected;
}

function boundPairSum(a, b, x) {
  return x + a + b;
}

function boundBetween(min, max, x) {
  return x > min && x < max;
}

function boundTripleEquals(a, b, x) {
  return x == a + b;
}

function boundTripleSum(a, b, c, x) {
  return x + a + b + c;
}

function boundWindow(min, mid, max, x) {
  return x > min && x < max && x != mid;
}

function boundQuadEquals(a, b, c, x) {
  return x == a + b + c;
}

function boundQuadSum(a, b, c, d, x) {
  return x + a + b + c + d;
}

function boundOuterWindow(min, low, high, max, x) {
  return x > min && x >= low && x < max && x != high;
}

function boundQuintEquals(a, b, c, d, x) {
  return x == a + b + c + d;
}

function boundQuintSum(a, b, c, d, e, x) {
  return x + a + b + c + d + e;
}

function boundSextEquals(a, b, c, d, e, x) {
  return x == a + b + c + d + e;
}

function boundSextSum(a, b, c, d, e, f, x) {
  return x + a + b + c + d + e + f;
}

function boundSeptEquals(a, b, c, d, e, f, x) {
  return x == a + b + c + d + e + f;
}

function boundSeptSum(a, b, c, d, e, f, g, x) {
  return x + a + b + c + d + e + f + g;
}

function boundOctEquals(a, b, c, d, e, f, g, x) {
  return x == a + b + c + d + e + f + g;
}

function boundOctSum(a, b, c, d, e, f, g, h, x) {
  return x + a + b + c + d + e + f + g + h;
}

function boundNonetEquals(a, b, c, d, e, f, g, h, x) {
  return x == a + b + c + d + e + f + g + h;
}

function boundNonetSum(a, b, c, d, e, f, g, h, i, x) {
  return x + a + b + c + d + e + f + g + h + i;
}

function boundDecetEquals(a, b, c, d, e, f, g, h, i, x) {
  return x == a + b + c + d + e + f + g + h + i;
}

function boundDecetSum(a, b, c, d, e, f, g, h, i, j, x) {
  return x + a + b + c + d + e + f + g + h + i + j;
}

function boundUndecEquals(a, b, c, d, e, f, g, h, i, j, x) {
  return x == a + b + c + d + e + f + g + h + i + j;
}

function boundUndecSum(a, b, c, d, e, f, g, h, i, j, k, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k;
}

function boundDuodecEquals(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x == a + b + c + d + e + f + g + h + i + j + k + l;
}

function boundDuodecSum(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k + l;
}


function main(args) {
  freshObject();
  freshSwitchCase();
  (() => "fresh-fn");
  new PlainCtorBox();
  new FreshReturnCtorBox();
  ({ name: "kimchi" });
  ({ answer: 41 }).answer;
  ({ label: "index" })["label"];
  ({ maybe: "opt-member" })?.maybe;
  ({ maybe: "opt-index" })?.["maybe"];
  "soup".length;
  [1, 2, 3];
  ` + "`soup${1}`" + `;
  "left" + "right";
  ~1;
  1n & 3n;
  1n === 1n;
  ("cmp-left" + "x") === ("cmp-right" + "y");
  !("not-left" + "right");
  ("and-left" + "x") && ("and-right" + "y");
  ("or-left" + "x") || ("or-right" + "y");
  typeof ("type" + "of");
  freshBox() instanceof FreshBox;
  ("ok" is "ok" | "error");
  ([1, "ok"] is [number, string]);
  ({ kind: "ok", value: 3 } is { kind: "ok", value: number } | { kind: "error", message: string });
  true ? ({ kind: "conditional" }) : ({ kind: "fallback" });
  null ?? ({ kind: "nullish" });
  (({ kind: "comma-left" }), ({ kind: "comma-right" }));
  freshInvokeObject.bind(null);
  freshInvokeObject.call(null);
  freshInvokeObject.apply(null, []);
  [1, 2].forEach((x) => 0);
  [1, 2].map((x) => x + 1);
  [1, 2].filter((x) => x > 0);
  [1, 2].find((x) => false);
  [1, 2].forEach(boundOffset.bind(null, 1));
  [1, 2].forEach(boundOffset.bind(null, 20));
  [1, 2].forEach(largeOffset);
  [1, 2].map(boundOffset.bind(null, 1));
  [1, 2].map(boundOffset.bind(null, 20));
  [1, 2].filter(largeOffset);
  [1, 2].filter(boundGreaterThan.bind(null, 0));
  [1, 2].filter(boundOffset.bind(null, 20));
  [1, 2].find(largeOffset);
  [1, 2].find(boundEquals.bind(null, 9));
  [1, 2].find(boundOffset.bind(null, 20));
  [1, 2].forEach(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 10, 10));
  [1, 2].filter(boundBetween.bind(null, 0, 3));
  [1, 2].find(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleEquals.bind(null, 4, 5));
  [1, 2].forEach(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].forEach(boundPairSum.bind(null, 10, 10));
  [1, 2].map(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].map(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundWindow.bind(null, 0, 1, 3));
  [1, 2].filter(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].find(boundQuadEquals.bind(null, 3, 4, 5));
  [1, 2].forEach(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].forEach(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].forEach(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].map(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].filter(boundOuterWindow.bind(null, 0, 1, 4, 3));
  [1, 2].filter(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].find(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].forEach(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].map(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].filter(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintEquals.bind(null, 30, 30, 30, 30, 30));
  [1, 2].find(boundQuintEquals.bind(null, 3, 4, 5, 6));
  [1, 2].forEach(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextEquals.bind(null, 40, 40, 40, 40, 40, 40));
  [1, 2].forEach(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptEquals.bind(null, 50, 50, 50, 50, 50, 50, 50));
  [1, 2].forEach(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctEquals.bind(null, 60, 60, 60, 60, 60, 60, 60, 60));
  [1, 2].forEach(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetEquals.bind(null, 70, 70, 70, 70, 70, 70, 70, 70, 70));
  [1, 2].forEach(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetEquals.bind(null, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80));
  [1, 2].forEach(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecEquals.bind(null, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90));
  [1, 2].forEach(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecEquals.bind(null, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100));
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "call ptr @jayess_template_string(") {
		t.Fatalf("expected template literal lowering in LLVM IR:\n%s", irText)
	}
	if !strings.Contains(irText, "call ptr @jayess_value_bind(") || !strings.Contains(irText, "call ptr @jayess_value_merge_bound_args(") {
		t.Fatalf("expected bind/call/apply lowering in LLVM IR:\n%s", irText)
	}
	if !strings.Contains(irText, "call ptr @jayess_value_bitwise_not(") {
		t.Fatalf("expected numeric bitwise lowering in LLVM IR:\n%s", irText)
	}
	if count := strings.Count(irText, "call void @jayess_value_free_unshared("); count < 6 {
		t.Fatalf("expected discarded fresh-expression cleanup calls in LLVM IR, got %d:\n%s", count, irText)
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

func TestCompileRejectsUsingBlockScopedVariableOutsideBlock(t *testing.T) {
	source := `
function main(args) {
  if (true) {
    var scoped = 1;
  }
  return scoped;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected block scope error")
	}
	if !strings.Contains(err.Error(), "unknown identifier scoped") {
		t.Fatalf("expected block-scoped identifier error, got: %v", err)
	}
}

func TestCompileReportsLexerDiagnosticWithSourceSpan(t *testing.T) {
	source := `
function main(args) {
  return @;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected lexer error")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Category != "lexer" || compileErr.Diagnostic.Code != "JY050" {
		t.Fatalf("expected lexer diagnostic, got %#v", compileErr.Diagnostic)
	}
	if compileErr.Diagnostic.Line != 3 || compileErr.Diagnostic.Column != 10 {
		t.Fatalf("expected lexer diagnostic span 3:10, got %d:%d", compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
	}
	if !strings.Contains(compileErr.Diagnostic.Message, "unexpected character") || !strings.Contains(compileErr.Diagnostic.Message, "@") {
		t.Fatalf("expected helpful lexer message, got %#v", compileErr.Diagnostic)
	}
}

func TestCompileReportsUnterminatedStringAsLexerDiagnostic(t *testing.T) {
	source := `
function main(args) {
  return "unterminated;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected lexer error")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Category != "lexer" {
		t.Fatalf("expected lexer category, got %#v", compileErr.Diagnostic)
	}
	if compileErr.Diagnostic.Line != 3 || compileErr.Diagnostic.Column != 10 {
		t.Fatalf("expected lexer diagnostic span 3:10, got %d:%d", compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
	}
	if compileErr.Diagnostic.Message != "unterminated string" {
		t.Fatalf("expected unterminated string message, got %#v", compileErr.Diagnostic)
	}
}

func TestCompileReportsParserDiagnosticWithSourceSpan(t *testing.T) {
	source := `
function main(args) {
  let value = 1;
  return value;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected parser error")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Category != "parse" || compileErr.Diagnostic.Code != "JY100" {
		t.Fatalf("expected parse diagnostic, got %#v", compileErr.Diagnostic)
	}
	if compileErr.Diagnostic.Line != 3 || compileErr.Diagnostic.Column != 3 {
		t.Fatalf("expected parser diagnostic span 3:3, got %d:%d", compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
	}
	if !strings.Contains(compileErr.Diagnostic.Message, "let is not supported") {
		t.Fatalf("expected helpful parser message, got %#v", compileErr.Diagnostic)
	}
}

func TestCompileSupportsProcessPathAndFsSurface(t *testing.T) {
	source := `
function main(args) {
  var cwd = process.cwd();
  var argv = process.argv();
  var platform = process.platform();
  var arch = process.arch();
  var hr = process.hrtime();
  var home = process.env("HOME");
  var shared = new SharedArrayBuffer(4);
  var sharedInts = new Int32Array(shared);
  var atomicLoaded = Atomics.load(sharedInts, 0);
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
  var dnsResult = dns.lookup("localhost");
  var dnsAll = dns.lookupAll("localhost");
  var dnsReverse = dns.reverse("127.0.0.1");
  var dnsSet = dns.setResolver({ hosts: { "kimchi.local": "127.0.0.9" }, reverse: { "127.0.0.9": "kimchi.local" } });
  var dnsClear = dns.clearResolver();
  var killed = childProcess.kill({ pid: 123, signal: "SIGTERM" });
  var w = worker.create(function(message) { return message; });
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
  console.log(cwd, argv, platform, arch, hr, home, atomicLoaded, sep, delimiter, normalized, resolved, relative, parsed, formatted, absolute, base, dir, ext, dnsResult, dnsAll, dnsReverse, dnsSet, dnsClear, killed, w, wrote, made, exists, text, stat, entries, copied, copiedDir, renamed, removed);
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
		"@jayess_std_process_hrtime",
		"@jayess_std_process_env",
		"@jayess_std_shared_array_buffer_new",
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
		"@jayess_std_dns_lookup",
		"@jayess_std_dns_lookup_all",
		"@jayess_std_dns_reverse",
		"@jayess_std_dns_set_resolver",
		"@jayess_std_dns_clear_resolver",
		"@jayess_std_child_process_kill",
		"@jayess_std_worker_create",
		"@jayess_atomics_load",
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

func TestCompileSupportsCryptoSurface(t *testing.T) {
	source := `
function main(args) {
  var bytes = crypto.randomBytes(16);
  var digest = crypto.hash("sha256", "kimchi");
  var mac = crypto.hmac("sha256", "secret", bytes);
  var same = crypto.secureCompare(digest, mac);
  var encrypted = crypto.encrypt({ algorithm: "aes-256-gcm", key: bytes, iv: bytes.slice(0, 12), data: "kimchi" });
  var decrypted = crypto.decrypt({ algorithm: "aes-256-gcm", key: bytes, iv: encrypted.iv, data: encrypted.ciphertext, tag: encrypted.tag });
  var pair = crypto.generateKeyPair({ type: "rsa", modulusLength: 2048 });
  var sealed = crypto.publicEncrypt({ algorithm: "rsa-oaep-sha256", key: pair.publicKey, data: "jjigae" });
  var opened = crypto.privateDecrypt({ algorithm: "rsa-oaep-sha256", key: pair.privateKey, data: sealed });
  var signature = crypto.sign({ algorithm: "rsa-pss-sha256", key: pair.privateKey, data: "kimchi" });
  var verified = crypto.verify({ algorithm: "rsa-pss-sha256", key: pair.publicKey, data: "kimchi", signature: signature });
  var gz = compression.gzip("kimchi");
  var gunzipped = compression.gunzip(gz);
  var df = compression.deflate("jjigae");
  var inflated = compression.inflate(df);
  var br = compression.brotli("mandu");
  var unbr = compression.unbrotli(br);
  var gzStream = compression.createGzipStream();
  var gunzipStream = compression.createGunzipStream();
  var deflateStream = compression.createDeflateStream();
  var inflateStream = compression.createInflateStream();
  var brotliStream = compression.createBrotliStream();
  var unbrotliStream = compression.createUnbrotliStream();
  console.log(bytes, digest, mac, same);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_std_crypto_random_bytes",
		"@jayess_std_crypto_hash",
		"@jayess_std_crypto_hmac",
		"@jayess_std_crypto_secure_compare",
		"@jayess_std_crypto_encrypt",
		"@jayess_std_crypto_decrypt",
		"@jayess_std_crypto_generate_key_pair",
		"@jayess_std_crypto_public_encrypt",
		"@jayess_std_crypto_private_decrypt",
		"@jayess_std_crypto_sign",
		"@jayess_std_crypto_verify",
		"@jayess_std_compression_gzip",
		"@jayess_std_compression_gunzip",
		"@jayess_std_compression_deflate",
		"@jayess_std_compression_inflate",
		"@jayess_std_compression_brotli",
		"@jayess_std_compression_unbrotli",
		"@jayess_std_compression_create_gzip_stream",
		"@jayess_std_compression_create_gunzip_stream",
		"@jayess_std_compression_create_deflate_stream",
		"@jayess_std_compression_create_inflate_stream",
		"@jayess_std_compression_create_brotli_stream",
		"@jayess_std_compression_create_unbrotli_stream",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected %s in LLVM IR, got:\n%s", symbol, irText)
		}
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn_greet(") {
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
	if !strings.Contains(irText, "call ptr @jayess_value_from_string") && !strings.Contains(irText, "call ptr @jayess_value_from_static_string") {
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn___jayess_lambda_0(") {
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn___jayess_lambda_0(") {
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

func TestCompileSupportsConditionalOperator(t *testing.T) {
	source := `
function main(args) {
  var label = args[0] ? "yes" : "no";
  print(label);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "cond.true") || !strings.Contains(llvmIR, "cond.false") {
		t.Fatalf("expected conditional operator to emit branch labels, got: %s", llvmIR)
	}
}

func TestCompileSupportsAutomaticSemicolonInsertion(t *testing.T) {
	source := `
function main(args) {
  var first = "kimchi"
  var second = "ramen"
  print(first + second)
  return 0
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "@jayess_print") {
		t.Fatalf("expected semicolonless program to compile, got: %s", llvmIR)
	}
}

func TestCompileSupportsReturnASI(t *testing.T) {
	source := `
function main(args) {
  return
  42;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "ret double 0.000000") {
		t.Fatalf("expected newline after return to end the statement, got: %s", llvmIR)
	}
}

func TestCompileRejectsThrowLineBreak(t *testing.T) {
	source := `
function main(args) {
  throw
  "boom";
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected line break after throw to be rejected")
	}
	if !strings.Contains(err.Error(), "line break or statement end is not allowed after throw") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileSupportsBigIntLiterals(t *testing.T) {
	source := `
function main(args) {
  const value = 123n;
  print(value);
  print(typeof 123n);
  print(123n === 123n);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "@jayess_value_from_bigint") {
		t.Fatalf("expected bigint literals to use runtime bigint boxing")
	}
	if !strings.Contains(llvmIR, "bigint") {
		t.Fatalf("expected bigint typeof support in generated IR")
	}
}

func TestCompileSupportsBitwiseOperators(t *testing.T) {
	source := `
function main(args) {
  const number = (5 & 3) | (8 ^ 1);
  const shifted = (number << 2) >> 1;
  const unsigned = -1 >>> 1;
  const big = (~5n) ^ (3n << 2n);
  print(number, shifted, unsigned, big);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	for _, helper := range []string{
		"@jayess_value_bitwise_and",
		"@jayess_value_bitwise_or",
		"@jayess_value_bitwise_xor",
		"@jayess_value_bitwise_shl",
		"@jayess_value_bitwise_shr",
		"@jayess_value_bitwise_ushr",
		"@jayess_value_bitwise_not",
	} {
		if !strings.Contains(llvmIR, helper) {
			t.Fatalf("expected bitwise operators to lower through %s", helper)
		}
	}
}

func TestCompileSupportsCommaExpressions(t *testing.T) {
	source := `
function main(args) {
  const value = (print("side effect"), 41 + 1);
  return value;
}
`

	p := parser.New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.Functions) == 0 || len(program.Functions[0].Body) == 0 {
		t.Fatalf("expected parsed function body")
	}
	decl, ok := program.Functions[0].Body[0].(*ast.VariableDecl)
	if !ok {
		t.Fatalf("expected first statement to be a variable declaration, got %T", program.Functions[0].Body[0])
	}
	if _, ok := decl.Value.(*ast.CommaExpression); !ok {
		t.Fatalf("expected variable initializer to parse as a comma expression, got %T", decl.Value)
	}

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "side effect") {
		t.Fatalf("expected comma expression to preserve left-hand side effects")
	}
}

func TestCompileSupportsDoWhile(t *testing.T) {
	source := `
function main(args) {
  var count = 0;
  do {
    count += 1;
  } while (count < 3);
  return count;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "dowhile.body") || !strings.Contains(llvmIR, "dowhile.cond") {
		t.Fatalf("expected do-while to lower through dedicated loop labels")
	}
}

func TestCompileSupportsLabeledStatements(t *testing.T) {
	source := `
function main(args) {
  var outer = 0;
  loop: while (outer < 5) {
    outer = outer + 1;
    block: {
      if (outer == 2) {
        break block;
      }
      if (outer == 4) {
        break loop;
      }
    }
    continue loop;
  }

  pick: switch (outer) {
    case 4:
      break pick;
    default:
      outer = outer + 10;
  }
  return outer;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	llvmIR := string(result.LLVMIR)
	if !strings.Contains(llvmIR, "label.end") || !strings.Contains(llvmIR, "switch.end") {
		t.Fatalf("expected labeled statement lowering in LLVM IR, got:\n%s", llvmIR)
	}
}

func TestCompileRejectsContinueToNonLoopLabel(t *testing.T) {
	source := `
function main(args) {
  block: {
    continue block;
  }
  return 0;
}
`

	_, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected Compile to reject continue to non-loop label")
	}
	if !strings.Contains(err.Error(), "unknown continue label block") {
		t.Fatalf("unexpected error: %v", err)
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn___jayess_lambda_0") {
		t.Fatalf("expected arrow function with destructured parameters to lower successfully")
	}
}

func TestCompileSupportsDestructuringRestElementsAndDefaults(t *testing.T) {
	source := `
function main(args) {
  const sourceObj = { b: 2 };
  const sourceArray = [undefined, 4, 5];
  const { a = 1, ...rest } = sourceObj;
  const [x = 3, ...tail] = sourceArray;
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

func TestCompileDirectLiteralDestructuringRestAvoidsGenericRestHelpers(t *testing.T) {
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
	if strings.Contains(irText, "call ptr @jayess_value_object_rest") || strings.Contains(irText, "call ptr @jayess_value_array_slice") {
		t.Fatalf("expected fresh literal destructuring rest to avoid generic object-rest/array-slice helpers, got:\n%s", irText)
	}
	if !strings.Contains(irText, "undefined") {
		t.Fatalf("expected destructuring defaults to keep undefined checks in lowered IR")
	}
}

func TestCompileDuplicateObjectKeyDestructuringRestFallsBackToGenericHelper(t *testing.T) {
	source := `
function main(args) {
  const { value, ...rest } = { value: 1, value: 2, extra: 3 };
  print(rest.extra);
  return value;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "call ptr @jayess_value_object_rest") {
		t.Fatalf("expected duplicate-key object literal destructuring to fall back to generic object-rest helper")
	}
}

func TestCompileArraySourceSpreadDestructuringFallsBackToGenericAccess(t *testing.T) {
	source := `
function main(args) {
  const [first] = [1, ...[2, 3]];
  return first;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected array-source-spread destructuring to fall back to generic indexed access")
	}
}

func TestCompileObjectSourceSpreadDestructuringFallsBackToGenericAccess(t *testing.T) {
	source := `
function main(args) {
  const { value } = { value: 1, ...{ extra: 2 } };
  return value;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "call ptr @jayess_value_get_member") {
		t.Fatalf("expected object-source-spread destructuring to fall back to generic member access")
	}
}

func TestCompileSupportsDestructuringInForLoopInit(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  for (var [i, limit] = [0, 3]; i < limit; i = i + 1) {
    total = total + i;
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected for-loop init destructuring to lower through indexed access helpers")
	}
}

func TestCompileSupportsDestructuringInForLoopUpdate(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  var i = 0;
  for (; i < 4; [i] = [i + 1]) {
    total = total + i;
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected for-loop update destructuring to lower through indexed access helpers")
	}
}

func TestCompileSupportsCompoundAssignmentInForLoopUpdate(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  for (var i = 0; i < 4; i += 1) {
    total = total + i;
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_add") {
		t.Fatalf("expected compound for-loop update to lower through add helper")
	}
}

func TestCompileSupportsDestructuringInForOfBinding(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  for (var [a, b] of [[1, 2], [3, 4]]) {
    total = total + a + b;
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_index") {
		t.Fatalf("expected for...of destructuring binding to lower through indexed access helpers")
	}
}

func TestCompileSupportsObjectDestructuringInForOfBinding(t *testing.T) {
	source := `
function main(args) {
  var total = 0;
  for (var { value = 1, ...rest } of [{ extra: 2 }, { value: 3, extra: 4 }]) {
    total = total + value + rest.extra;
  }
  return total;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_get_member") || !strings.Contains(irText, "@jayess_value_object_rest") {
		t.Fatalf("expected for...of object destructuring binding to lower through member/object-rest helpers")
	}
	if !strings.Contains(irText, "undefined") {
		t.Fatalf("expected for...of object destructuring defaults to keep undefined checks in lowered IR")
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
	if !strings.Contains(irText, "@jayess_fn___jayess_lambda_0") {
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
	if !strings.Contains(irText, "@jayess_fn___jayess_lambda_0") || !strings.Contains(irText, "@jayess_fn___jayess_lambda_1") {
		t.Fatalf("expected nested lowered closure helpers in LLVM IR")
	}
}

func TestCompilePreservesSharedCapturedMutation(t *testing.T) {
	source := `
function main(args) {
  var count = 0;
  var inc = () => {
    count = count + 1;
    return count;
  };
  var read = () => count;
  return inc() + read();
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if strings.Count(irText, "@jayess_value_get_member") < 2 || !strings.Contains(irText, "@jayess_value_set_member") {
		t.Fatalf("expected closure cell accesses to lower through object member helpers, got:\n%s", irText)
	}
	if !strings.Contains(irText, "c\"count\\00\"") || !strings.Contains(irText, "c\"value\\00\"") {
		t.Fatalf("expected closure cell lowering to access captured variable and cell value members, got:\n%s", irText)
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn_Counter__read") {
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
	if !strings.Contains(irText, "@jayess_fn_Base__read") {
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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn_Child__total") {
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
	if !strings.Contains(irText, "@jayess_fn___jayess_dispatch__sound__0") || !strings.Contains(irText, "@jayess_value_function_ptr") {
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
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected function.call to use first-class invoke helpers")
	}
	if strings.Contains(irText, "@.str.") && strings.Contains(irText, "c\"call\\00\"") && strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected function.call lowering to target the callable value directly")
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
	if strings.Contains(irText, "c\"apply\\00\"") && strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected function.apply lowering to target the callable value directly")
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
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_function_ptr") {
		t.Fatalf("expected function.bind result to remain a first-class callable value")
	}
	if strings.Contains(irText, "c\"bind\\00\"") && strings.Contains(irText, "@jayess_value_get_member") {
		t.Fatalf("expected function.bind lowering to target the callable value directly")
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

func TestCompileSupportsSymbol(t *testing.T) {
	source := `
function main(args) {
  var left = Symbol("kimchi");
  var right = Symbol("kimchi");
  print(typeof left);
  print(left == left);
  print(left == right);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_symbol") {
		t.Fatalf("expected Symbol() to lower to runtime symbol helper")
	}
	if !strings.Contains(irText, "@jayess_value_typeof") {
		t.Fatalf("expected typeof on symbols to use runtime typeof helper")
	}
	if !strings.Contains(irText, "@jayess_value_eq") {
		t.Fatalf("expected symbol equality to use runtime equality helper")
	}
}

func TestCompileSupportsSymbolPropertyKeys(t *testing.T) {
	source := `
function main(args) {
  var sym = Symbol("k");
  var obj = { [sym]: 1, plain: 2 };
  print(obj[sym]);
  print(Object.hasOwn(obj, sym));
  print(Object.getOwnPropertySymbols(obj));
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_set_computed_member") {
		t.Fatalf("expected symbol-keyed object literals to use computed member runtime helper")
	}
	if !strings.Contains(irText, "@jayess_value_object_symbols") {
		t.Fatalf("expected Object.getOwnPropertySymbols to lower to runtime symbol-key helper")
	}
}

func TestCompileSupportsSymbolRegistryAndWellKnownSymbols(t *testing.T) {
	source := `
function main(args) {
  var left = Symbol.for("kimchi");
  var right = Symbol.for("kimchi");
  var key = Symbol.keyFor(left);
  var iter = Symbol.iterator;
  print(left == right);
  print(key);
  print(typeof iter);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, want := range []string{
		"@jayess_std_symbol_for",
		"@jayess_std_symbol_key_for",
		"@jayess_std_symbol_iterator",
	} {
		if !strings.Contains(irText, want) {
			t.Fatalf("expected generated IR to contain %s", want)
		}
	}
}

func TestCompileSupportsTypedArrays(t *testing.T) {
	source := `
function main(args) {
  var buffer = new ArrayBuffer(16);
  var a = new Int8Array(buffer);
  var b = new Uint16Array(buffer);
  var c = new Float32Array(2);
  print(a[0]);
  print(b[0]);
  print(c[0]);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, want := range []string{
		"@jayess_std_int8_array_new",
		"@jayess_std_uint16_array_new",
		"@jayess_std_float32_array_new",
	} {
		if !strings.Contains(irText, want) {
			t.Fatalf("expected generated IR to contain %s", want)
		}
	}
}

func TestCompileSupportsIterableProtocol(t *testing.T) {
	source := `
function main(args) {
  var iterable = {
    [Symbol.iterator]: function() {
      return {
        next: function() {
          return { value: 1, done: true };
        }
      };
    }
  };
  var values = Array.from(iterable);
  for (var ch of "ab") {
    print(ch);
  }
  return values.length;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_iterable_values") {
		t.Fatalf("expected iterable protocol lowering to use runtime iterable helper")
	}
}

func TestCompileSupportsGenerators(t *testing.T) {
	source := `
function* values() {
  yield 1;
  yield 2;
  return 99;
}

function main(args) {
  var make = function*() {
    yield 3;
  };
  var iter = values();
  var other = make();
  print(iter.next().value);
  print(other.next().value);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_iterator_from") {
		t.Fatalf("expected generator lowering to return iterator values, got:\n%s", irText)
	}
}

func TestCompileSupportsAsyncGenerators(t *testing.T) {
	source := `
async function* values() {
  yield await Promise.resolve(1);
  yield 2;
}

function main(args) {
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_std_async_iterator_from") || !strings.Contains(irText, "@jayess_await") {
		t.Fatalf("expected async generator lowering to use async iterator and await helpers, got:\n%s", irText)
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
	if !strings.Contains(string(result.LLVMIR), "switch.case") || !strings.Contains(string(result.LLVMIR), "switch.end") {
		t.Fatalf("expected switch to lower through dedicated switch labels")
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

func TestCompileSupportsObjectSpread(t *testing.T) {
	source := `
function main(args) {
  var base = { second: "b", third: "c" };
  var obj = { first: "a", ...base, fourth: "d" };
  print(obj.first);
  print(obj.fourth);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@jayess_value_object_assign") {
		t.Fatalf("expected object spread to lower through runtime object assign helper")
	}
}

func TestCompileSupportsObjectLiteralAccessors(t *testing.T) {
	source := `
function main(args) {
  var hidden = 10;
  var obj = {
    get value() {
      return hidden;
    },
    set value(next) {
      hidden = next;
    }
  };
  obj.value = 42;
  print(obj.value);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, want := range []string{
		"@jayess_value_get_member",
		"@jayess_value_set_member",
		"__jayess_get_value",
		"__jayess_set_value",
	} {
		if !strings.Contains(irText, want) {
			t.Fatalf("expected object literal accessors to include %q in generated IR", want)
		}
	}
}

func TestCompileSupportsClassAccessors(t *testing.T) {
	source := `
class Counter {
  constructor() {
    this._value = 1;
  }

  get value() {
    return this._value;
  }

  set value(next) {
    this._value = next;
  }

  static get label() {
    return "counter";
  }

  static set label(next) {
    print(next);
  }
}

function main(args) {
  var counter = new Counter();
  counter.value = 7;
  print(counter.value);
  Counter.label = "updated";
  print(Counter.label);
  return 0;
}
`

	result, err := Compile(source, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, want := range []string{
		"Counter__accessor__get__value",
		"Counter__accessor__set__value",
		"Counter__static_accessor__get__label",
		"Counter__static_accessor__set__label",
		"__jayess_get_value",
		"__jayess_set_value",
	} {
		if !strings.Contains(irText, want) {
			t.Fatalf("expected class accessors to include %q in generated IR", want)
		}
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
	case *ast.DoWhileStatement:
		for _, child := range stmt.Body {
			collectIdentifiersFromStatement(child, out)
		}
		collectIdentifiersFromExpression(stmt.Condition, out)
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
	case *ast.BlockStatement:
		for _, child := range stmt.Body {
			collectIdentifiersFromStatement(child, out)
		}
	case *ast.SwitchStatement:
		collectIdentifiersFromExpression(stmt.Discriminant, out)
		for _, switchCase := range stmt.Cases {
			collectIdentifiersFromExpression(switchCase.Test, out)
			for _, child := range switchCase.Consequent {
				collectIdentifiersFromStatement(child, out)
			}
		}
		for _, child := range stmt.Default {
			collectIdentifiersFromStatement(child, out)
		}
	case *ast.LabeledStatement:
		collectIdentifiersFromStatement(stmt.Statement, out)
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
	case *ast.CommaExpression:
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
	case *ast.DoWhileStatement:
		return functionBodyContainsIdentifier(stmt.Body, name) || expressionContainsIdentifier(stmt.Condition, name)
	case *ast.ForStatement:
		return (stmt.Init != nil && statementContainsIdentifier(stmt.Init, name)) ||
			(stmt.Condition != nil && expressionContainsIdentifier(stmt.Condition, name)) ||
			(stmt.Update != nil && statementContainsIdentifier(stmt.Update, name)) ||
			functionBodyContainsIdentifier(stmt.Body, name)
	case *ast.BlockStatement:
		return functionBodyContainsIdentifier(stmt.Body, name)
	case *ast.SwitchStatement:
		if expressionContainsIdentifier(stmt.Discriminant, name) || functionBodyContainsIdentifier(stmt.Default, name) {
			return true
		}
		for _, switchCase := range stmt.Cases {
			if expressionContainsIdentifier(switchCase.Test, name) || functionBodyContainsIdentifier(switchCase.Consequent, name) {
				return true
			}
		}
		return false
	case *ast.LabeledStatement:
		return statementContainsIdentifier(stmt.Statement, name)
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
	case *ast.CommaExpression:
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
