package lexer

type TokenType string

const (
	TokenEOF       TokenType = "EOF"
	TokenIllegal   TokenType = "ILLEGAL"
	TokenFunction  TokenType = "FUNCTION"
	TokenExtern    TokenType = "EXTERN"
	TokenImport    TokenType = "IMPORT"
	TokenNative    TokenType = "NATIVE"
	TokenVar       TokenType = "VAR"
	TokenLet       TokenType = "LET"
	TokenConst     TokenType = "CONST"
	TokenPrivate   TokenType = "PRIVATE"
	TokenPublic    TokenType = "PUBLIC"
	TokenReturn    TokenType = "RETURN"
	TokenIf        TokenType = "IF"
	TokenElse      TokenType = "ELSE"
	TokenWhile     TokenType = "WHILE"
	TokenFor       TokenType = "FOR"
	TokenBreak     TokenType = "BREAK"
	TokenContinue  TokenType = "CONTINUE"
	TokenTrue      TokenType = "TRUE"
	TokenFalse     TokenType = "FALSE"
	TokenNull      TokenType = "NULL"
	TokenUndefined TokenType = "UNDEFINED"
	TokenIdent     TokenType = "IDENT"
	TokenNumber    TokenType = "NUMBER"
	TokenString    TokenType = "STRING"
	TokenLParen    TokenType = "("
	TokenRParen    TokenType = ")"
	TokenLBrace    TokenType = "{"
	TokenRBrace    TokenType = "}"
	TokenLBracket  TokenType = "["
	TokenRBracket  TokenType = "]"
	TokenSemicolon TokenType = ";"
	TokenAssign    TokenType = "="
	TokenEq        TokenType = "=="
	TokenNe        TokenType = "!="
	TokenBang      TokenType = "!"
	TokenLt        TokenType = "<"
	TokenLte       TokenType = "<="
	TokenGt        TokenType = ">"
	TokenGte       TokenType = ">="
	TokenComma     TokenType = ","
	TokenColon     TokenType = ":"
	TokenDot       TokenType = "."
	TokenPlus      TokenType = "+"
	TokenMinus     TokenType = "-"
	TokenStar      TokenType = "*"
	TokenSlash     TokenType = "/"
	TokenAnd       TokenType = "&&"
	TokenOr        TokenType = "||"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}
