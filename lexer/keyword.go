package lexer

import "unicode"

func isIdentifierStart(ch rune) bool {
	return ch == '_' || ch == '$' || unicode.IsLetter(ch)
}

func isIdentifierPart(ch rune) bool {
	return isIdentifierStart(ch) || unicode.IsDigit(ch)
}

func lookupIdent(literal string) TokenType {
	switch literal {
	case "function":
		return TokenFunction
	case "class":
		return TokenClass
	case "enum":
		return TokenEnum
	case "extends":
		return TokenExtends
	case "extern":
		return TokenExtern
	case "import":
		return TokenImport
	case "export":
		return TokenExport
	case "default":
		return TokenDefault
	case "native":
		return TokenNative
	case "static":
		return TokenStatic
	case "new":
		return TokenNew
	case "this":
		return TokenThis
	case "super":
		return TokenSuper
	case "var":
		return TokenVar
	case "let":
		return TokenLet
	case "const":
		return TokenConst
	case "private":
		return TokenPrivate
	case "public":
		return TokenPublic
	case "return":
		return TokenReturn
	case "if":
		return TokenIf
	case "else":
		return TokenElse
	case "do":
		return TokenDo
	case "while":
		return TokenWhile
	case "with":
		return TokenWith
	case "for":
		return TokenFor
	case "of":
		return TokenOf
	case "in":
		return TokenIn
	case "switch":
		return TokenSwitch
	case "case":
		return TokenCase
	case "break":
		return TokenBreak
	case "continue":
		return TokenContinue
	case "delete":
		return TokenDelete
	case "debugger":
		return TokenDebugger
	case "try":
		return TokenTry
	case "catch":
		return TokenCatch
	case "finally":
		return TokenFinally
	case "throw":
		return TokenThrow
	case "await":
		return TokenAwait
	case "async":
		return TokenAsync
	case "yield":
		return TokenYield
	case "typeof":
		return TokenTypeof
	case "void":
		return TokenVoid
	case "instanceof":
		return TokenInstanceof
	case "is":
		return TokenIs
	case "true":
		return TokenTrue
	case "false":
		return TokenFalse
	case "null":
		return TokenNull
	case "undefined":
		return TokenUndefined
	default:
		return TokenIdent
	}
}
