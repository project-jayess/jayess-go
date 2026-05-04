package parser

import "jayess-go/lexer"

func isObjectPropertyNameToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent,
		lexer.TokenString,
		lexer.TokenNumber,
		lexer.TokenTrue,
		lexer.TokenFalse,
		lexer.TokenNull,
		lexer.TokenUndefined,
		lexer.TokenFunction,
		lexer.TokenClass,
		lexer.TokenEnum,
		lexer.TokenExtends,
		lexer.TokenExtern,
		lexer.TokenImport,
		lexer.TokenExport,
		lexer.TokenDefault,
		lexer.TokenNative,
		lexer.TokenStatic,
		lexer.TokenNew,
		lexer.TokenThis,
		lexer.TokenSuper,
		lexer.TokenVar,
		lexer.TokenLet,
		lexer.TokenConst,
		lexer.TokenPrivate,
		lexer.TokenPublic,
		lexer.TokenReturn,
		lexer.TokenIf,
		lexer.TokenElse,
		lexer.TokenDo,
		lexer.TokenWhile,
		lexer.TokenWith,
		lexer.TokenFor,
		lexer.TokenOf,
		lexer.TokenIn,
		lexer.TokenSwitch,
		lexer.TokenCase,
		lexer.TokenBreak,
		lexer.TokenContinue,
		lexer.TokenDelete,
		lexer.TokenDebugger,
		lexer.TokenTry,
		lexer.TokenCatch,
		lexer.TokenFinally,
		lexer.TokenThrow,
		lexer.TokenAwait,
		lexer.TokenAsync,
		lexer.TokenYield,
		lexer.TokenTypeof,
		lexer.TokenVoid,
		lexer.TokenInstanceof,
		lexer.TokenIs:
		return true
	default:
		return false
	}
}
