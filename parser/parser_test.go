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

func TestParseProgramSupportsDestructuringInForOfBinding(t *testing.T) {
	source := `
function main(args) {
  for (var { value = 1, ...rest } of items) {
    print(value);
    print(rest.extra);
  }
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if len(program.Functions) != 1 || len(program.Functions[0].Body) == 0 {
		t.Fatalf("expected parsed function body")
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForOfStatement)
	if !ok {
		t.Fatalf("expected first statement to be for...of, got %T", program.Functions[0].Body[0])
	}
	if loop.Name == "" {
		t.Fatalf("expected synthetic loop binding name")
	}
	if len(loop.Body) == 0 {
		t.Fatalf("expected loop body")
	}
	binding, ok := loop.Body[0].(*ast.DestructuringDecl)
	if !ok {
		t.Fatalf("expected for...of body to begin with destructuring decl, got %T", loop.Body[0])
	}
	objectPattern, ok := binding.Pattern.(*ast.ObjectPattern)
	if !ok {
		t.Fatalf("expected object destructuring pattern, got %T", binding.Pattern)
	}
	if len(objectPattern.Properties) != 1 || objectPattern.Properties[0].Key != "value" || objectPattern.Rest != "rest" {
		t.Fatalf("unexpected object pattern: %#v", objectPattern)
	}
	identifier, ok := binding.Value.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected destructuring source identifier, got %T", binding.Value)
	}
	if identifier.Name != loop.Name {
		t.Fatalf("expected destructuring source %q to match loop binding %q", identifier.Name, loop.Name)
	}
}

func TestParseProgramRejectsDestructuringInForInBinding(t *testing.T) {
	source := `
function main(args) {
  for (var [key] in obj) {
    return 0;
  }
  return 0;
}
`

	p := New(lexer.New(source))
	if _, err := p.ParseProgram(); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestParseProgramChoosesNonConflictingForOfDestructuringTempName(t *testing.T) {
	source := `
function main(args) {
  for (var [value] of items) {
    const __jayess_foreach_0 = 1;
    print(__jayess_foreach_0);
    print(value);
  }
  return 0;
}
`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForOfStatement)
	if !ok {
		t.Fatalf("expected first statement to be for...of, got %T", program.Functions[0].Body[0])
	}
	if loop.Name == "__jayess_foreach_0" {
		t.Fatalf("expected synthetic loop binding name to avoid body identifier collision")
	}
	binding, ok := loop.Body[0].(*ast.DestructuringDecl)
	if !ok {
		t.Fatalf("expected for...of body to begin with destructuring decl, got %T", loop.Body[0])
	}
	identifier, ok := binding.Value.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected destructuring source identifier, got %T", binding.Value)
	}
	if identifier.Name != loop.Name {
		t.Fatalf("expected destructuring source %q to match loop binding %q", identifier.Name, loop.Name)
	}
}

func TestParseProgramPreservesForOfDestructuringPatternSourcePosition(t *testing.T) {
	source := `function main(args) {
  for (const [value] of items) {
    print(value);
  }
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForOfStatement)
	if !ok {
		t.Fatalf("expected first statement to be for...of, got %T", program.Functions[0].Body[0])
	}
	binding, ok := loop.Body[0].(*ast.DestructuringDecl)
	if !ok {
		t.Fatalf("expected for...of body to begin with destructuring decl, got %T", loop.Body[0])
	}
	if binding.Pos.Line != 2 || binding.Pos.Column != 14 {
		t.Fatalf("expected destructuring binding position 2:14, got %d:%d", binding.Pos.Line, binding.Pos.Column)
	}
	identifier, ok := binding.Value.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected destructuring source identifier, got %T", binding.Value)
	}
	if identifier.Pos.Line != binding.Pos.Line || identifier.Pos.Column != binding.Pos.Column {
		t.Fatalf("expected synthetic source identifier to track binding position %d:%d, got %d:%d", binding.Pos.Line, binding.Pos.Column, identifier.Pos.Line, identifier.Pos.Column)
	}
}

func TestParseProgramPreservesForVariableInitSourcePosition(t *testing.T) {
	source := `function main(args) {
  for (const value = 1; value < 2; value = value + 1) {
    return value;
  }
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected first statement to be for loop, got %T", program.Functions[0].Body[0])
	}
	init, ok := loop.Init.(*ast.VariableDecl)
	if !ok {
		t.Fatalf("expected for init to be variable decl, got %T", loop.Init)
	}
	if init.Pos.Line != 2 || init.Pos.Column != 8 {
		t.Fatalf("expected for-init variable position 2:8, got %d:%d", init.Pos.Line, init.Pos.Column)
	}
}

func TestParseProgramPreservesForDestructuringInitSourcePosition(t *testing.T) {
	source := `function main(args) {
  for (const [value] = [1]; value < 2; value = value + 1) {
    return value;
  }
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected first statement to be for loop, got %T", program.Functions[0].Body[0])
	}
	init, ok := loop.Init.(*ast.DestructuringDecl)
	if !ok {
		t.Fatalf("expected for init to be destructuring decl, got %T", loop.Init)
	}
	if init.Pos.Line != 2 || init.Pos.Column != 8 {
		t.Fatalf("expected for-init destructuring position 2:8, got %d:%d", init.Pos.Line, init.Pos.Column)
	}
}

func TestParseProgramPreservesAssignmentSourcePosition(t *testing.T) {
	source := `function main(args) {
  value = 1;
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	stmt, ok := program.Functions[0].Body[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected assignment statement, got %T", program.Functions[0].Body[0])
	}
	if stmt.Pos.Line != 2 || stmt.Pos.Column != 3 {
		t.Fatalf("expected assignment position 2:3, got %d:%d", stmt.Pos.Line, stmt.Pos.Column)
	}
}

func TestParseProgramPreservesDestructuringAssignmentSourcePosition(t *testing.T) {
	source := `function main(args) {
  [value] = [1];
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	stmt, ok := program.Functions[0].Body[0].(*ast.DestructuringAssignment)
	if !ok {
		t.Fatalf("expected destructuring assignment, got %T", program.Functions[0].Body[0])
	}
	if stmt.Pos.Line != 2 || stmt.Pos.Column != 3 {
		t.Fatalf("expected destructuring assignment position 2:3, got %d:%d", stmt.Pos.Line, stmt.Pos.Column)
	}
}

func TestParseProgramPreservesForUpdateAssignmentSourcePosition(t *testing.T) {
	source := `function main(args) {
  for (; value < 2; value = value + 1) {
    return value;
  }
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected first statement to be for loop, got %T", program.Functions[0].Body[0])
	}
	update, ok := loop.Update.(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected for update to be assignment, got %T", loop.Update)
	}
	if update.Pos.Line != 2 || update.Pos.Column != 21 {
		t.Fatalf("expected for-update assignment position 2:21, got %d:%d", update.Pos.Line, update.Pos.Column)
	}
	if update.Operator != ast.AssignmentAssign {
		t.Fatalf("expected for-update assignment operator %q, got %q", ast.AssignmentAssign, update.Operator)
	}
}

func TestParseProgramSupportsCompoundAssignmentInForUpdate(t *testing.T) {
	source := `function main(args) {
  for (; value < 2; value += 1) {
    return value;
  }
}`

	p := New(lexer.New(source))
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	loop, ok := program.Functions[0].Body[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected first statement to be for loop, got %T", program.Functions[0].Body[0])
	}
	update, ok := loop.Update.(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected for update to be assignment, got %T", loop.Update)
	}
	if update.Operator != ast.AssignmentAddAssign {
		t.Fatalf("expected for-update assignment operator %q, got %q", ast.AssignmentAddAssign, update.Operator)
	}
}
