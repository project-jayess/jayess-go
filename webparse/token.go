package webparse

import "strings"

type tokenType string

const (
	tokenText    tokenType = "text"
	tokenStart   tokenType = "start"
	tokenEnd     tokenType = "end"
	tokenComment tokenType = "comment"
	tokenPI      tokenType = "processing"
	tokenCDATA   tokenType = "cdata"
)

type token struct {
	typ        tokenType
	raw        string
	name       string
	attrs      []Attribute
	selfClosed bool
	span       Span
}

func tokenizeMarkup(input string, file string, xml bool) ([]token, []Diagnostic) {
	var tokens []token
	var diagnostics []Diagnostic
	for offset := 0; offset < len(input); {
		span := spanAt(input, file, offset)
		if input[offset] != '<' {
			next := strings.IndexByte(input[offset:], '<')
			if next < 0 {
				next = len(input) - offset
			}
			tokens = append(tokens, token{typ: tokenText, raw: input[offset : offset+next], span: span})
			offset += next
			continue
		}
		switch {
		case strings.HasPrefix(input[offset:], "<!--"):
			end := strings.Index(input[offset+4:], "-->")
			if end < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unterminated comment", Span: span, Recoverable: !xml})
				end = len(input) - offset - 4
			}
			raw := input[offset+4 : offset+4+end]
			tokens = append(tokens, token{typ: tokenComment, raw: raw, span: span})
			offset += 4 + end
			if strings.HasPrefix(input[offset:], "-->") {
				offset += 3
			}
		case strings.HasPrefix(input[offset:], "<![CDATA["):
			end := strings.Index(input[offset+9:], "]]>")
			if end < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unterminated cdata", Span: span, Recoverable: !xml})
				end = len(input) - offset - 9
			}
			raw := input[offset+9 : offset+9+end]
			tokens = append(tokens, token{typ: tokenCDATA, raw: raw, span: span})
			offset += 9 + end
			if strings.HasPrefix(input[offset:], "]]>") {
				offset += 3
			}
		case strings.HasPrefix(input[offset:], "<?"):
			end := strings.Index(input[offset+2:], "?>")
			if end < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unterminated processing instruction", Span: span, Recoverable: !xml})
				end = len(input) - offset - 2
			}
			raw := strings.TrimSpace(input[offset+2 : offset+2+end])
			tokens = append(tokens, token{typ: tokenPI, raw: raw, span: span})
			offset += 2 + end
			if strings.HasPrefix(input[offset:], "?>") {
				offset += 2
			}
		default:
			end := strings.IndexByte(input[offset:], '>')
			if end < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unterminated tag", Span: span, Recoverable: !xml})
				end = len(input) - offset - 1
			}
			body := strings.TrimSpace(input[offset+1 : offset+end])
			tokens = append(tokens, parseTag(body, span, xml, &diagnostics))
			offset += end + 1
		}
	}
	return tokens, diagnostics
}

func parseTag(body string, span Span, xml bool, diagnostics *[]Diagnostic) token {
	if strings.HasPrefix(body, "/") {
		return token{typ: tokenEnd, name: strings.TrimSpace(body[1:]), span: span}
	}
	selfClosed := strings.HasSuffix(body, "/")
	body = strings.TrimSpace(strings.TrimSuffix(body, "/"))
	name, rest, _ := strings.Cut(body, " ")
	return token{typ: tokenStart, name: name, attrs: parseAttrs(rest, span, xml, diagnostics), selfClosed: selfClosed, span: span}
}

func parseAttrs(input string, span Span, xml bool, diagnostics *[]Diagnostic) []Attribute {
	var attrs []Attribute
	for input = strings.TrimSpace(input); input != ""; input = strings.TrimSpace(input) {
		name := readAttrName(input)
		if name == "" {
			break
		}
		input = strings.TrimSpace(input[len(name):])
		attr := Attribute{Name: name, Boolean: true, Span: span}
		if strings.HasPrefix(input, "=") {
			input = strings.TrimSpace(input[1:])
			attr.Value, input = readAttrValue(input)
			attr.Boolean = false
		} else if xml {
			*diagnostics = append(*diagnostics, Diagnostic{Message: "xml attribute missing value", Span: span})
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

func readAttrName(input string) string {
	for index, r := range input {
		if r == '=' || r == '/' || r == '>' || r == '"' || r == '\'' || r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return input[:index]
		}
	}
	return input
}

func readAttrValue(input string) (string, string) {
	if input == "" {
		return "", ""
	}
	if input[0] == '"' || input[0] == '\'' {
		quote := input[0]
		end := strings.IndexByte(input[1:], quote)
		if end < 0 {
			return input[1:], ""
		}
		return input[1 : end+1], input[end+2:]
	}
	for index, r := range input {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return input[:index], input[index:]
		}
	}
	return input, ""
}

func spanAt(input string, file string, offset int) Span {
	span := Span{File: file, Line: 1, Column: 1, Offset: offset}
	for index, r := range input {
		if index >= offset {
			break
		}
		if r == '\n' {
			span.Line++
			span.Column = 1
		} else {
			span.Column++
		}
	}
	return span
}
