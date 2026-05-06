# Lexer

The lexer converts source text into a token stream with source positions.

## Responsibilities

- read source text deterministically
- identify supported Jayess tokens
- preserve byte, line, and column information for diagnostics
- report invalid characters or malformed literals

## Token Consumers

The parser consumes lexer tokens to build AST nodes. Diagnostics and source text
helpers use token positions to point users at the relevant source span.

## Text Handling

The lexer is expected to work with compiler-sized source files. Source text APIs
track byte offsets and line/column positions consistently for diagnostics.

## Example Tokens

```js
const total = 1 + 2;
```

This source is expected to produce tokens for `const`, an identifier, `=`,
integer literals, `+`, `;`, and end-of-file, each with stable source positions.
