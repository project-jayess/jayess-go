package parser

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
)

func TestParseProgramSupportsTypeAliasAndAsAssertion(t *testing.T) {
	source := `
type Count = number;

function main(args) {
  var value: Count = 1 as Count;
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 1 {
		t.Fatalf("expected 1 type alias, got %d", len(program.TypeAliases))
	}
	if program.TypeAliases[0].Name != "Count" || program.TypeAliases[0].Target != "number" {
		t.Fatalf("unexpected alias: %#v", program.TypeAliases[0])
	}
	if len(program.Functions) != 1 || len(program.Functions[0].Body) == 0 {
		t.Fatalf("expected parsed function body")
	}
	decl, ok := program.Functions[0].Body[0].(*ast.VariableDecl)
	if !ok {
		t.Fatalf("expected first statement to be variable declaration, got %T", program.Functions[0].Body[0])
	}
	if decl.TypeAnnotation != "Count" {
		t.Fatalf("expected type annotation Count, got %q", decl.TypeAnnotation)
	}
	cast, ok := decl.Value.(*ast.CastExpression)
	if !ok {
		t.Fatalf("expected cast expression, got %T", decl.Value)
	}
	if cast.TypeAnnotation != "Count" {
		t.Fatalf("expected cast annotation Count, got %q", cast.TypeAnnotation)
	}
}

func TestParseProgramRejectsInvalidTypeAlias(t *testing.T) {
	source := `
type Count = ;

function main(args) {
  return 0;
}
`

	p := New(lexer.New(source))
	if _, err := p.ParseProgram(); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestParseProgramSupportsInterfacesAndStructuredTypes(t *testing.T) {
	source := `
interface User {
  readonly id: number;
  name: string;
  age?: number;
  [key: string]: string;
}

type Pair = [number, string];

function main(args) {
  const user: User = { id: 1, name: "kimchi" };
  const pair: Pair = [1, "ramen"];
  const mapper: (number, string) => boolean = (count: number, label: string): boolean => count > 0;
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 2 {
		t.Fatalf("expected 2 type aliases, got %d", len(program.TypeAliases))
	}
	if program.TypeAliases[0].Name != "User" || program.TypeAliases[0].Target != "{readonly id:number;name:string;age?:number;[key:string]:string;}" {
		t.Fatalf("unexpected interface alias: %#v", program.TypeAliases[0])
	}
	if program.TypeAliases[1].Name != "Pair" || program.TypeAliases[1].Target != "[number,string]" {
		t.Fatalf("unexpected tuple alias: %#v", program.TypeAliases[1])
	}
	decl, ok := program.Functions[0].Body[2].(*ast.VariableDecl)
	if !ok {
		t.Fatalf("expected mapper variable declaration, got %T", program.Functions[0].Body[2])
	}
	if decl.TypeAnnotation != "(number,string)=>boolean" {
		t.Fatalf("expected callable variable type, got %q", decl.TypeAnnotation)
	}
}

func TestParseProgramSupportsLiteralAndUnionTypes(t *testing.T) {
	source := `
type Status = "ok" | "error";
type Tagged = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  const status: Status = "ok";
  const tagged: Tagged = { kind: "ok", value: 1 };
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 2 {
		t.Fatalf("expected 2 type aliases, got %d", len(program.TypeAliases))
	}
	if program.TypeAliases[0].Target != "\"ok\"|\"error\"" {
		t.Fatalf("unexpected union alias: %#v", program.TypeAliases[0])
	}
	if program.TypeAliases[1].Target != "{kind:\"ok\",value:number}|{kind:\"error\",message:string}" {
		t.Fatalf("unexpected tagged union alias: %#v", program.TypeAliases[1])
	}
}

func TestParseProgramSupportsIntersectionTypes(t *testing.T) {
	source := `
type Combined = { id: number } & { name: string };

function main(args) {
  const value: Combined = { id: 1, name: "kimchi" };
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 1 {
		t.Fatalf("expected 1 type alias, got %d", len(program.TypeAliases))
	}
	if program.TypeAliases[0].Target != "{id:number}&{name:string}" {
		t.Fatalf("unexpected intersection alias: %#v", program.TypeAliases[0])
	}
}

func TestParseProgramSupportsEnums(t *testing.T) {
	source := `
enum Status {
  Ok,
  Error = 3,
  Ready = "ready",
}

function main(args) {
  const status: Status = Status.Ok;
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 1 {
		t.Fatalf("expected 1 generated type alias, got %d", len(program.TypeAliases))
	}
	if program.TypeAliases[0].Name != "Status" || program.TypeAliases[0].Target != "0|3|\"ready\"" {
		t.Fatalf("unexpected enum alias: %#v", program.TypeAliases[0])
	}
	if len(program.Globals) != 1 || program.Globals[0].Name != "Status" {
		t.Fatalf("expected generated enum object global, got %#v", program.Globals)
	}
}

func TestParseProgramSupportsRuntimeTypeChecks(t *testing.T) {
	source := `
function main(args) {
  const value = 1;
  const ok = value is number | string;
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.Functions) != 1 || len(program.Functions[0].Body) == 0 {
		t.Fatalf("expected parsed main body, got %#v", program.Functions)
	}
	decl, ok := program.Functions[0].Body[1].(*ast.VariableDecl)
	if !ok {
		t.Fatalf("expected runtime type check variable declaration, got %T", program.Functions[0].Body[1])
	}
	check, ok := decl.Value.(*ast.TypeCheckExpression)
	if !ok {
		t.Fatalf("expected TypeCheckExpression, got %T", decl.Value)
	}
	if check.TypeAnnotation != "number|string" {
		t.Fatalf("unexpected runtime type annotation: %q", check.TypeAnnotation)
	}
}

func TestParseProgramSupportsGenericAliasesAndInterfaces(t *testing.T) {
	source := `
type Box<T extends number | string> = { value: T };
interface Pair<T> {
  left: T,
  right: T,
}

function main(args) {
  const box: Box<number> = { value: 1 };
  const pair: Pair<string> = { left: "a", right: "b" };
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.TypeAliases) != 2 {
		t.Fatalf("expected 2 type aliases, got %d", len(program.TypeAliases))
	}
	if len(program.TypeAliases[0].TypeParams) != 1 || program.TypeAliases[0].TypeParams[0].Constraint != "number|string" {
		t.Fatalf("unexpected generic alias params: %#v", program.TypeAliases[0].TypeParams)
	}
	if len(program.TypeAliases[1].TypeParams) != 1 {
		t.Fatalf("unexpected generic interface params: %#v", program.TypeAliases[1].TypeParams)
	}
}

func TestParseProgramRejectsImplicitEnumAfterStringMember(t *testing.T) {
	source := `
enum Status {
  Ready = "ready",
  Done,
}

function main(args) {
  return 0;
}
`

	p := New(lexer.New(source))
	if _, err := p.ParseProgram(); err == nil {
		t.Fatalf("expected parse error")
	}
}
