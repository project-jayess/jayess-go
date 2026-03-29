package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	classHeaderPattern         = regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)(?:\s+extends\s+([A-Za-z_][A-Za-z0-9_]*))?\s*\{\s*$`)
	methodHeaderPattern        = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{\s*$`)
	privateMethodHeaderPattern = regexp.MustCompile(`^\s*#([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{\s*$`)

	staticMethodHeaderPattern        = regexp.MustCompile(`^\s*static\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{\s*$`)
	privateStaticMethodHeaderPattern = regexp.MustCompile(`^\s*static\s+#([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{\s*$`)

	staticFieldPattern        = regexp.MustCompile(`^\s*static\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*(.+))?;\s*$`)
	privateStaticFieldPattern = regexp.MustCompile(`^\s*static\s+#([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*(.+))?;\s*$`)
	instanceFieldPattern      = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+);\s*$`)
	privateFieldPattern       = regexp.MustCompile(`^\s*#([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*(.+))?;\s*$`)

	functionScopePattern     = regexp.MustCompile(`^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	constructorAssignPattern = regexp.MustCompile(`^\s*(var|const)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:new\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	assignNewPattern         = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:new\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

	memberCallPattern          = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	selfCallPattern            = regexp.MustCompile(`\b__self\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	thisCallPattern            = regexp.MustCompile(`\bthis\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	thisPrivateCallPattern     = regexp.MustCompile(`\bthis\.#([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	thisPropertyPattern        = regexp.MustCompile(`\bthis\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	thisPrivatePropertyPattern = regexp.MustCompile(`\bthis\.#([A-Za-z_][A-Za-z0-9_]*)\b`)
	superCallPattern           = regexp.MustCompile(`\bsuper\s*\(`)
	superMethodPattern         = regexp.MustCompile(`\bsuper\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	superPropertyPattern       = regexp.MustCompile(`\bsuper\.([A-Za-z_][A-Za-z0-9_]*)\b`)

	staticThisCallPattern            = regexp.MustCompile(`\bthis\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	staticThisPrivateCallPattern     = regexp.MustCompile(`\bthis\.#([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	staticThisPropertyPattern        = regexp.MustCompile(`\bthis\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	staticThisPrivatePropertyPattern = regexp.MustCompile(`\bthis\.#([A-Za-z_][A-Za-z0-9_]*)\b`)
	staticSuperMethodPattern         = regexp.MustCompile(`\bsuper\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	staticSuperPropertyPattern       = regexp.MustCompile(`\bsuper\.([A-Za-z_][A-Za-z0-9_]*)\b`)
)

type classInfo struct {
	name                 string
	base                 string
	constructorParams    string
	methods              map[string]bool
	privateMethods       map[string]bool
	staticMethods        map[string]bool
	privateStaticMethods map[string]bool
	staticFields         map[string]bool
	privateStaticFields  map[string]bool
	instanceFields       []fieldDecl
	privateFields        []fieldDecl
}

type fieldDecl struct {
	name  string
	value string
}

type scopeInfo struct {
	classes    map[string]string
	isFunction bool
}

func transpileClasses(source string) (string, error) {
	lines := strings.Split(source, "\n")
	transformedLines, classes, err := lowerClassDeclarations(lines)
	if err != nil {
		return "", err
	}
	rewritten := rewriteClassUsages(transformedLines, classes)
	return strings.Join(rewritten, "\n"), nil
}

func lowerClassDeclarations(lines []string) ([]string, map[string]classInfo, error) {
	var out []string
	classes := map[string]classInfo{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		matches := classHeaderPattern.FindStringSubmatch(line)
		if matches == nil {
			out = append(out, line)
			continue
		}

		info, emitted, nextIndex, err := parseClass(lines, i, matches[1], matches[2], classes)
		if err != nil {
			return nil, nil, err
		}
		classes[info.name] = info
		out = append(out, emitted...)
		i = nextIndex
	}
	return out, classes, nil
}

func parseClass(lines []string, start int, className, baseName string, classes map[string]classInfo) (classInfo, []string, int, error) {
	info := classInfo{
		name:                 className,
		base:                 baseName,
		methods:              map[string]bool{},
		privateMethods:       map[string]bool{},
		staticMethods:        map[string]bool{},
		privateStaticMethods: map[string]bool{},
		staticFields:         map[string]bool{},
		privateStaticFields:  map[string]bool{},
	}

	var emitted []string
	depth := 1
	i := start + 1
	constructorFound := false

	for ; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "}" {
			depth--
			if depth == 0 {
				break
			}
			continue
		}

		switch {
		case privateStaticFieldPattern.MatchString(line):
			matches := privateStaticFieldPattern.FindStringSubmatch(line)
			info.privateStaticFields[matches[1]] = true
			emitted = append(emitted, emitStaticField(className, matches[1], defaultFieldValue(matches[2]), true)...)
			continue
		case staticFieldPattern.MatchString(line):
			matches := staticFieldPattern.FindStringSubmatch(line)
			info.staticFields[matches[1]] = true
			emitted = append(emitted, emitStaticField(className, matches[1], defaultFieldValue(matches[2]), false)...)
			continue
		case privateStaticMethodHeaderPattern.MatchString(line):
			matches := privateStaticMethodHeaderPattern.FindStringSubmatch(line)
			bodyLines, nextIndex, err := collectMethodBody(lines, i)
			if err != nil {
				return classInfo{}, nil, 0, err
			}
			i = nextIndex
			info.privateStaticMethods[matches[1]] = true
			emitted = append(emitted, emitStaticMethod(info, classes, matches[1], strings.TrimSpace(matches[2]), bodyLines, true)...)
			continue
		case staticMethodHeaderPattern.MatchString(line):
			matches := staticMethodHeaderPattern.FindStringSubmatch(line)
			bodyLines, nextIndex, err := collectMethodBody(lines, i)
			if err != nil {
				return classInfo{}, nil, 0, err
			}
			i = nextIndex
			info.staticMethods[matches[1]] = true
			emitted = append(emitted, emitStaticMethod(info, classes, matches[1], strings.TrimSpace(matches[2]), bodyLines, false)...)
			continue
		case privateFieldPattern.MatchString(line):
			matches := privateFieldPattern.FindStringSubmatch(line)
			info.privateFields = append(info.privateFields, fieldDecl{name: matches[1], value: defaultFieldValue(matches[2])})
			continue
		case instanceFieldPattern.MatchString(line):
			matches := instanceFieldPattern.FindStringSubmatch(line)
			info.instanceFields = append(info.instanceFields, fieldDecl{name: matches[1], value: strings.TrimSpace(matches[2])})
			continue
		case privateMethodHeaderPattern.MatchString(line):
			matches := privateMethodHeaderPattern.FindStringSubmatch(line)
			bodyLines, nextIndex, err := collectMethodBody(lines, i)
			if err != nil {
				return classInfo{}, nil, 0, err
			}
			i = nextIndex
			if matches[1] == "constructor" {
				constructorFound = true
				info.constructorParams = strings.TrimSpace(matches[2])
				emitted = append(emitted, emitConstructor(info, classes, strings.TrimSpace(matches[2]), bodyLines)...)
				continue
			}
			info.privateMethods[matches[1]] = true
			emitted = append(emitted, emitMethod(info, classes, matches[1], strings.TrimSpace(matches[2]), bodyLines, true)...)
			continue
		case methodHeaderPattern.MatchString(line):
			matches := methodHeaderPattern.FindStringSubmatch(line)
			bodyLines, nextIndex, err := collectMethodBody(lines, i)
			if err != nil {
				return classInfo{}, nil, 0, err
			}
			i = nextIndex

			if matches[1] == "constructor" {
				constructorFound = true
				info.constructorParams = strings.TrimSpace(matches[2])
				emitted = append(emitted, emitConstructor(info, classes, strings.TrimSpace(matches[2]), bodyLines)...)
				continue
			}
			info.methods[matches[1]] = true
			emitted = append(emitted, emitMethod(info, classes, matches[1], strings.TrimSpace(matches[2]), bodyLines, false)...)
			continue
		default:
			return classInfo{}, nil, 0, fmt.Errorf("unsupported class syntax in %s: %s", className, trimmed)
		}
	}

	if !constructorFound {
		emitted = append(defaultConstructorLines(info), emitted...)
	}

	if i >= len(lines) || strings.TrimSpace(lines[i]) != "}" {
		return classInfo{}, nil, 0, fmt.Errorf("unterminated class %s", className)
	}
	return info, emitted, i, nil
}

func collectMethodBody(lines []string, headerIndex int) ([]string, int, error) {
	var body []string
	depth := 1
	for i := headerIndex + 1; i < len(lines); i++ {
		line := lines[i]
		depth += strings.Count(line, "{")
		depth -= strings.Count(line, "}")
		if depth == 0 {
			return body, i, nil
		}
		body = append(body, line)
	}
	return nil, 0, fmt.Errorf("unterminated method body")
}

func emitConstructor(info classInfo, classes map[string]classInfo, params string, body []string) []string {
	rewritten := rewriteMethodBody(info, classes, body, false)
	rewritten = injectInstanceFieldInitializers(info, rewritten)

	initExpr := "{}"
	if info.base != "" {
		initExpr = "undefined"
	}

	lines := []string{
		fmt.Sprintf("function %s(%s) {", info.name, params),
		fmt.Sprintf("  var __self = %s;", initExpr),
	}
	lines = append(lines, rewritten...)
	lines = append(lines, "  return __self;", "}", "")
	return lines
}

func defaultConstructorLines(info classInfo) []string {
	lines := []string{
		fmt.Sprintf("function %s() {", info.name),
	}
	if info.base != "" {
		lines = append(lines, fmt.Sprintf("  var __self = %s();", info.base))
	} else {
		lines = append(lines, "  var __self = {};")
	}
	lines = append(lines, instanceFieldInitializerLines(info)...)
	lines = append(lines, "  return __self;", "}", "")
	return lines
}

func emitMethod(info classInfo, classes map[string]classInfo, methodName, params string, body []string, private bool) []string {
	fullParams := "__self"
	if strings.TrimSpace(params) != "" {
		fullParams += ", " + params
	}
	lines := []string{
		fmt.Sprintf("function %s(%s) {", methodSymbol(info.name, methodName, private), fullParams),
	}
	lines = append(lines, rewriteMethodBody(info, classes, body, false)...)
	lines = append(lines, "}", "")
	return lines
}

func emitStaticMethod(info classInfo, classes map[string]classInfo, methodName, params string, body []string, private bool) []string {
	lines := []string{
		fmt.Sprintf("function %s(%s) {", staticMemberSymbol(info.name, methodName, private), params),
	}
	lines = append(lines, rewriteMethodBody(info, classes, body, true)...)
	lines = append(lines, "}", "")
	return lines
}

func emitStaticField(className, fieldName, value string, private bool) []string {
	return []string{
		fmt.Sprintf("var %s = %s;", staticMemberSymbol(className, fieldName, private), value),
		"",
	}
}

func rewriteMethodBody(info classInfo, classes map[string]classInfo, body []string, isStatic bool) []string {
	var out []string
	for _, line := range body {
		rewritten := line
		if isStatic {
			rewritten = rewriteStaticBodyLine(info, classes, rewritten)
		} else {
			rewritten = rewriteInstanceBodyLine(info, classes, rewritten)
		}
		out = append(out, strings.ReplaceAll(rewritten, ", )", ")"))
	}
	return out
}

func rewriteInstanceBodyLine(info classInfo, classes map[string]classInfo, line string) string {
	rewritten := line

	rewritten = thisPrivateCallPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := thisPrivateCallPattern.FindStringSubmatch(match)
		if len(parts) != 2 || !info.privateMethods[parts[1]] {
			return match
		}
		return methodSymbol(info.name, parts[1], true) + "(__self, "
	})
	rewritten = thisPrivatePropertyPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := thisPrivatePropertyPattern.FindStringSubmatch(match)
		if len(parts) != 2 || !hasPrivateField(info, parts[1]) {
			return match
		}
		return "__self." + privateFieldStorage(info.name, parts[1])
	})

	if info.base != "" {
		rewritten = superCallPattern.ReplaceAllString(rewritten, "__self = "+info.base+"(")
		rewritten = superMethodPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
			parts := superMethodPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			owner := lookupInstanceMethodOwner(classes, info.base, parts[1])
			if owner == "" {
				return match
			}
			return methodSymbol(owner, parts[1], false) + "(__self, "
		})
		rewritten = superPropertyPattern.ReplaceAllString(rewritten, "__self.$1")
	}

	rewritten = thisCallPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := thisCallPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if info.privateMethods[parts[1]] {
			return methodSymbol(info.name, parts[1], true) + "(__self, "
		}
		if owner := lookupInstanceMethodOwnerForClass(classes, info, parts[1]); owner != "" {
			return methodSymbol(owner, parts[1], false) + "(__self, "
		}
		return "__self." + parts[1] + "("
	})

	rewritten = selfCallPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := selfCallPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if info.privateMethods[parts[1]] {
			return methodSymbol(info.name, parts[1], true) + "(__self, "
		}
		if owner := lookupInstanceMethodOwnerForClass(classes, info, parts[1]); owner != "" {
			return methodSymbol(owner, parts[1], false) + "(__self, "
		}
		return match
	})

	rewritten = thisPropertyPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := thisPropertyPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if hasPrivateField(info, parts[1]) {
			return "__self." + privateFieldStorage(info.name, parts[1])
		}
		return "__self." + parts[1]
	})

	return rewritten
}

func rewriteStaticBodyLine(info classInfo, classes map[string]classInfo, line string) string {
	rewritten := line

	rewritten = staticThisPrivateCallPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := staticThisPrivateCallPattern.FindStringSubmatch(match)
		if len(parts) != 2 || !info.privateStaticMethods[parts[1]] {
			return match
		}
		return staticMemberSymbol(info.name, parts[1], true) + "("
	})
	rewritten = staticThisPrivatePropertyPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := staticThisPrivatePropertyPattern.FindStringSubmatch(match)
		if len(parts) != 2 || !info.privateStaticFields[parts[1]] {
			return match
		}
		return staticMemberSymbol(info.name, parts[1], true)
	})

	if info.base != "" {
		rewritten = staticSuperMethodPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
			parts := staticSuperMethodPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			owner := lookupStaticMethodOwner(classes, info.base, parts[1])
			if owner == "" {
				return match
			}
			return staticMemberSymbol(owner, parts[1], false) + "("
		})
		rewritten = staticSuperPropertyPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
			parts := staticSuperPropertyPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			if owner := lookupStaticFieldOwner(classes, info.base, parts[1]); owner != "" {
				return staticMemberSymbol(owner, parts[1], false)
			}
			return match
		})
	}

	rewritten = staticThisCallPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := staticThisCallPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if info.privateStaticMethods[parts[1]] {
			return staticMemberSymbol(info.name, parts[1], true) + "("
		}
		if owner := lookupStaticMethodOwnerForClass(classes, info, parts[1]); owner != "" {
			return staticMemberSymbol(owner, parts[1], false) + "("
		}
		return staticMemberSymbol(info.name, parts[1], false) + "("
	})

	rewritten = staticThisPropertyPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := staticThisPropertyPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if info.privateStaticFields[parts[1]] {
			return staticMemberSymbol(info.name, parts[1], true)
		}
		if owner := lookupStaticFieldOwnerForClass(classes, info, parts[1]); owner != "" {
			return staticMemberSymbol(owner, parts[1], false)
		}
		return staticMemberSymbol(info.name, parts[1], false)
	})

	return rewritten
}

func injectInstanceFieldInitializers(info classInfo, body []string) []string {
	initializers := instanceFieldInitializerLines(info)
	if len(initializers) == 0 {
		return body
	}
	if info.base == "" {
		return append(initializers, body...)
	}

	var out []string
	inserted := false
	for _, line := range body {
		out = append(out, line)
		if !inserted && strings.Contains(line, "__self = "+info.base+"(") {
			out = append(out, initializers...)
			inserted = true
		}
	}
	if inserted {
		return out
	}

	fallback := []string{
		fmt.Sprintf("  if (!__self) { __self = %s(); }", info.base),
	}
	fallback = append(fallback, initializers...)
	return append(fallback, out...)
}

func instanceFieldInitializerLines(info classInfo) []string {
	var lines []string
	for _, field := range info.instanceFields {
		lines = append(lines, fmt.Sprintf("  __self.%s = %s;", field.name, field.value))
	}
	for _, field := range info.privateFields {
		lines = append(lines, fmt.Sprintf("  __self.%s = %s;", privateFieldStorage(info.name, field.name), field.value))
	}
	return lines
}

func rewriteClassUsages(lines []string, classes map[string]classInfo) []string {
	var out []string
	scopeStack := []scopeInfo{{classes: map[string]string{}}}

	for _, original := range lines {
		line := original
		for className := range classes {
			line = strings.ReplaceAll(line, "new "+className+"(", className+"(")
			for fieldName, owner := range staticFieldOwners(classes, className) {
				line = strings.ReplaceAll(line, className+"."+fieldName, staticMemberSymbol(owner, fieldName, false))
			}
			for methodName, owner := range staticMethodOwners(classes, className) {
				line = strings.ReplaceAll(line, className+"."+methodName+"(", staticMemberSymbol(owner, methodName, false)+"(")
			}
		}

		scope := &scopeStack[len(scopeStack)-1]
		if matches := constructorAssignPattern.FindStringSubmatch(line); matches != nil {
			if _, ok := classes[matches[3]]; ok {
				scope.classes[matches[2]] = matches[3]
			}
		} else if matches := assignNewPattern.FindStringSubmatch(line); matches != nil {
			if _, ok := classes[matches[2]]; ok {
				assignClass(scopeStack, matches[1], matches[2])
			}
		}

		line = rewriteInstanceMethodCalls(line, scopeStack, classes)
		line = strings.ReplaceAll(line, ", )", ")")
		out = append(out, line)

		if functionScopePattern.MatchString(strings.TrimSpace(line)) {
			scopeStack = append(scopeStack, scopeInfo{classes: map[string]string{}, isFunction: true})
		}
		pushCount := strings.Count(line, "{")
		if functionScopePattern.MatchString(strings.TrimSpace(line)) && pushCount > 0 {
			pushCount--
		}
		for j := 0; j < pushCount; j++ {
			scopeStack = append(scopeStack, scopeInfo{classes: map[string]string{}})
		}
		popCount := strings.Count(line, "}")
		for j := 0; j < popCount && len(scopeStack) > 1; j++ {
			scopeStack = scopeStack[:len(scopeStack)-1]
		}
	}
	return out
}

func rewriteInstanceMethodCalls(line string, scopes []scopeInfo, classes map[string]classInfo) string {
	bindings := visibleClassBindings(scopes)
	return memberCallPattern.ReplaceAllStringFunc(line, func(match string) string {
		parts := memberCallPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		className, ok := bindings[parts[1]]
		if !ok {
			return match
		}
		owner := lookupInstanceMethodOwner(classes, className, parts[2])
		if owner == "" {
			return match
		}
		return methodSymbol(owner, parts[2], false) + "(" + parts[1] + ", "
	})
}

func visibleClassBindings(scopes []scopeInfo) map[string]string {
	result := map[string]string{}
	for i := 0; i < len(scopes); i++ {
		for name, className := range scopes[i].classes {
			result[name] = className
		}
	}
	return result
}

func assignClass(scopes []scopeInfo, name, className string) {
	for i := len(scopes) - 1; i >= 0; i-- {
		if _, ok := scopes[i].classes[name]; ok {
			scopes[i].classes[name] = className
			return
		}
	}
	scopes[len(scopes)-1].classes[name] = className
}

func instanceMethodOwners(classes map[string]classInfo, className string) map[string]string {
	result := map[string]string{}
	var visit func(string)
	visit = func(name string) {
		info, ok := classes[name]
		if !ok {
			return
		}
		if info.base != "" {
			visit(info.base)
		}
		for methodName := range info.methods {
			result[methodName] = name
		}
	}
	visit(className)
	return result
}

func staticMethodOwners(classes map[string]classInfo, className string) map[string]string {
	result := map[string]string{}
	var visit func(string)
	visit = func(name string) {
		info, ok := classes[name]
		if !ok {
			return
		}
		if info.base != "" {
			visit(info.base)
		}
		for methodName := range info.staticMethods {
			result[methodName] = name
		}
	}
	visit(className)
	return result
}

func staticFieldOwners(classes map[string]classInfo, className string) map[string]string {
	result := map[string]string{}
	var visit func(string)
	visit = func(name string) {
		info, ok := classes[name]
		if !ok {
			return
		}
		if info.base != "" {
			visit(info.base)
		}
		for fieldName := range info.staticFields {
			result[fieldName] = name
		}
	}
	visit(className)
	return result
}

func lookupInstanceMethodOwner(classes map[string]classInfo, className, methodName string) string {
	return instanceMethodOwners(classes, className)[methodName]
}

func lookupInstanceMethodOwnerForClass(classes map[string]classInfo, info classInfo, methodName string) string {
	if info.methods[methodName] {
		return info.name
	}
	if info.base == "" {
		return ""
	}
	return lookupInstanceMethodOwner(classes, info.base, methodName)
}

func lookupStaticMethodOwner(classes map[string]classInfo, className, methodName string) string {
	return staticMethodOwners(classes, className)[methodName]
}

func lookupStaticMethodOwnerForClass(classes map[string]classInfo, info classInfo, methodName string) string {
	if info.staticMethods[methodName] {
		return info.name
	}
	if info.base == "" {
		return ""
	}
	return lookupStaticMethodOwner(classes, info.base, methodName)
}

func lookupStaticFieldOwner(classes map[string]classInfo, className, fieldName string) string {
	return staticFieldOwners(classes, className)[fieldName]
}

func lookupStaticFieldOwnerForClass(classes map[string]classInfo, info classInfo, fieldName string) string {
	if info.staticFields[fieldName] {
		return info.name
	}
	if info.base == "" {
		return ""
	}
	return lookupStaticFieldOwner(classes, info.base, fieldName)
}

func hasPrivateField(info classInfo, fieldName string) bool {
	for _, field := range info.privateFields {
		if field.name == fieldName {
			return true
		}
	}
	return false
}

func methodSymbol(className, methodName string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, methodName)
	}
	return fmt.Sprintf("%s__%s", className, methodName)
}

func staticMemberSymbol(className, name string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, name)
	}
	return fmt.Sprintf("%s__%s", className, name)
}

func privateFieldStorage(className, fieldName string) string {
	return fmt.Sprintf("__jayess_private__%s__%s", className, fieldName)
}

func defaultFieldValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "undefined"
	}
	return value
}
