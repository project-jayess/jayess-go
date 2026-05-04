package webparse

import "strings"

type Stylesheet struct {
	Rules       []CSSRule
	Diagnostics []Diagnostic
}

type CSSRule struct {
	AtRule       bool
	Prelude      string
	Selectors    []string
	Declarations []Declaration
	Span         Span
}

type Declaration struct {
	Property string
	Value    string
	Span     Span
}

func ParseCSSStylesheet(input string) Stylesheet {
	clean, diagnostics := stripCSSComments(input)
	var rules []CSSRule
	for offset := 0; offset < len(clean); {
		for offset < len(clean) && isCSSSpace(clean[offset]) {
			offset++
		}
		if offset >= len(clean) {
			break
		}
		if strings.HasPrefix(clean[offset:], "@") {
			nextOpen := strings.IndexByte(clean[offset:], '{')
			nextSemi := strings.IndexByte(clean[offset:], ';')
			if nextSemi >= 0 && (nextOpen < 0 || nextSemi < nextOpen) {
				prelude := strings.TrimSpace(clean[offset : offset+nextSemi])
				rules = append(rules, CSSRule{AtRule: true, Prelude: prelude, Span: spanAt(clean, "", offset)})
				offset += nextSemi + 1
				continue
			}
		}
		open := strings.IndexByte(clean[offset:], '{')
		if open < 0 {
			break
		}
		open += offset
		close := strings.IndexByte(clean[open+1:], '}')
		if close < 0 {
			diagnostics = append(diagnostics, Diagnostic{Message: "unterminated css rule", Span: spanAt(clean, "", open), Recoverable: true})
			break
		}
		close += open + 1
		prelude := strings.TrimSpace(clean[offset:open])
		body := clean[open+1 : close]
		rule := CSSRule{
			Prelude:      prelude,
			AtRule:       strings.HasPrefix(prelude, "@"),
			Selectors:    parseSelectors(prelude),
			Declarations: parseDeclarations(body, clean, open+1),
			Span:         spanAt(clean, "", offset),
		}
		rules = append(rules, rule)
		offset = close + 1
	}
	return Stylesheet{Rules: rules, Diagnostics: diagnostics}
}

func isCSSSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func parseSelectors(prelude string) []string {
	if strings.HasPrefix(prelude, "@") {
		return nil
	}
	parts := strings.Split(prelude, ",")
	selectors := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			selectors = append(selectors, trimmed)
		}
	}
	return selectors
}

func parseDeclarations(body string, full string, baseOffset int) []Declaration {
	parts := strings.Split(body, ";")
	declarations := make([]Declaration, 0, len(parts))
	cursor := 0
	for _, part := range parts {
		name, value, ok := strings.Cut(part, ":")
		if !ok {
			cursor += len(part) + 1
			continue
		}
		declarations = append(declarations, Declaration{
			Property: strings.TrimSpace(name),
			Value:    strings.TrimSpace(value),
			Span:     spanAt(full, "", baseOffset+cursor),
		})
		cursor += len(part) + 1
	}
	return declarations
}

func stripCSSComments(input string) (string, []Diagnostic) {
	var diagnostics []Diagnostic
	var out strings.Builder
	for offset := 0; offset < len(input); {
		if strings.HasPrefix(input[offset:], "/*") {
			end := strings.Index(input[offset+2:], "*/")
			if end < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unterminated css comment", Span: spanAt(input, "", offset), Recoverable: true})
				break
			}
			offset += end + 4
			continue
		}
		out.WriteByte(input[offset])
		offset++
	}
	return out.String(), diagnostics
}

func SerializeCSS(sheet Stylesheet) string {
	var out strings.Builder
	for _, rule := range sheet.Rules {
		out.WriteString(rule.Prelude)
		out.WriteString("{")
		for index, declaration := range rule.Declarations {
			if index > 0 {
				out.WriteString(";")
			}
			out.WriteString(declaration.Property)
			out.WriteString(":")
			out.WriteString(declaration.Value)
		}
		out.WriteString("}")
	}
	return out.String()
}
