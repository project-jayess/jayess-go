package resolver

import (
	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func parseProgramSource(source []byte) (*ast.Program, error) {
	return parser.New(lexer.New(string(source))).ParseProgram()
}
