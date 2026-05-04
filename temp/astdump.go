package main
import (
  "fmt"
  "jayess-go/ast"
  "jayess-go/lexer"
  "jayess-go/parser"
)
func main(){
 p,err:=parser.New(lexer.New(`function main() { var code = 1; const chosen = (code++, []) ?? {}; if (typeof chosen === "object") { return code * 10 + 1; } return 1; }`)).ParseProgram(); if err!=nil{panic(err)}
 fn:=p.Statements[0].(*ast.FunctionDecl)
 vd:=fn.Body[1].(*ast.VariableDecl)
 fmt.Printf("%T %#v\n", vd.Value, vd.Value)
}
