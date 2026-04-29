package codegen

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"jayess-go/ir"
)

func validateClassLayout(module *ir.Module) error {
	functions := map[string]ir.Function{}
	for _, fn := range module.Functions {
		functions[fn.Name] = fn
	}
	globals := map[string]bool{}
	for _, global := range module.Globals {
		globals[global.Name] = true
	}
	classes := map[string]ir.ClassDecl{}
	for _, classDecl := range module.Classes {
		classes[classDecl.Name] = classDecl
	}
	for _, classDecl := range module.Classes {
		if classDecl.SuperClass != "" {
			if _, ok := classes[classDecl.SuperClass]; !ok {
				return fmt.Errorf("codegen class validation failed: class %s extends unknown class %s", classDecl.Name, classDecl.SuperClass)
			}
		}
		if _, ok := functions[classDecl.Name]; !ok {
			return fmt.Errorf("codegen class validation failed: missing lowered constructor for class %s", classDecl.Name)
		}
		for _, field := range classDecl.Fields {
			if !field.Static {
				continue
			}
			name := classStaticSymbol(classDecl.Name, field.Name, field.Private)
			if !globals[name] {
				return fmt.Errorf("codegen class validation failed: missing lowered static field %s for class %s", field.Name, classDecl.Name)
			}
		}
		for _, method := range classDecl.Methods {
			if method.IsConstructor {
				continue
			}
			name := classMethodSymbol(classDecl.Name, method.Name, method.Private, method.Static, method.IsGetter, method.IsSetter)
			fn, ok := functions[name]
			if !ok {
				return fmt.Errorf("codegen class validation failed: missing lowered method %s for class %s", method.Name, classDecl.Name)
			}
			expectedParams := method.ParamCount
			if !method.Static {
				expectedParams++
			}
			if len(fn.Params) != expectedParams {
				return fmt.Errorf("codegen class validation failed: lowered method %s for class %s has %d params, expected %d", method.Name, classDecl.Name, len(fn.Params), expectedParams)
			}
		}
	}
	return nil
}

func emitClassMetadata(buf *bytes.Buffer, classes []ir.ClassDecl) {
	if len(classes) == 0 {
		return
	}
	fmt.Fprintf(buf, "; class metadata\n")
	for _, classDecl := range classes {
		super := "none"
		if classDecl.SuperClass != "" {
			super = classDecl.SuperClass
		}
		fmt.Fprintf(buf, "; class %s extends %s\n", classDecl.Name, super)
		fmt.Fprintf(buf, ";   fields=%d methods=%d\n", len(classDecl.Fields), len(classDecl.Methods))
		for _, field := range classDecl.Fields {
			kind := "instance"
			if field.Static {
				kind = "static"
			}
			if field.Private {
				kind += " private"
			}
			initFlag := "noinit"
			if field.HasInitializer {
				initFlag = "init"
			}
			fmt.Fprintf(buf, ";   field %s [%s %s]\n", field.Name, kind, initFlag)
		}
		for _, method := range classDecl.Methods {
			kind := "instance"
			if method.Static {
				kind = "static"
			}
			if method.Private {
				kind += " private"
			}
			if method.IsConstructor {
				kind = "constructor"
			}
			fmt.Fprintf(buf, ";   method %s [%s params=%d]\n", method.Name, kind, method.ParamCount)
		}
	}
	buf.WriteString("\n")
}

func classMethodSymbol(className, methodName string, private, static, getter, setter bool) string {
	if getter || setter {
		kind := "set"
		if getter {
			kind = "get"
		}
		if static {
			return fmt.Sprintf("%s__static_accessor__%s__%s", className, kind, methodName)
		}
		return fmt.Sprintf("%s__accessor__%s__%s", className, kind, methodName)
	}
	if static {
		return classStaticSymbol(className, methodName, private)
	}
	if private {
		return fmt.Sprintf("%s__private__%s", className, methodName)
	}
	return fmt.Sprintf("%s__%s", className, methodName)
}

func classStaticSymbol(className, name string, private bool) string {
	if private {
		return fmt.Sprintf("%s__private__%s", className, name)
	}
	return fmt.Sprintf("%s__%s", className, name)
}

func stackFrameLabel(fn ir.Function) string {
	if fn.Line > 0 && fn.Column > 0 {
		return fmt.Sprintf("%s (%d:%d)", fn.Name, fn.Line, fn.Column)
	}
	return fn.Name
}

func emittedFunctionName(name string) string {
	if name == "main" {
		return "jayess_user_main"
	}
	return "jayess_fn_" + name
}

func emitFunctionSourceComment(buf *bytes.Buffer, fn ir.Function, loweredName string) {
	if fn.Line > 0 && fn.Column > 0 {
		fmt.Fprintf(buf, "; source function %s at %d:%d\n", fn.Name, fn.Line, fn.Column)
		fmt.Fprintf(buf, "; debug frame %s\n", stackFrameLabel(fn))
	} else {
		fmt.Fprintf(buf, "; source function %s\n", fn.Name)
	}
	if loweredName != fn.Name {
		fmt.Fprintf(buf, "; lowered symbol @%s\n", loweredName)
	}
}

func escapeLLVMMetadataString(text string) string {
	replacer := strings.NewReplacer(
		`\\`, `\5C`,
		`\`, `\5C`,
		`"`, `\22`,
		"\n", `\0A`,
		"\r", `\0D`,
		"\t", `\09`,
	)
	return replacer.Replace(text)
}

func buildDebugMetadataState(module *ir.Module) *debugMetadataState {
	enabled := false
	for _, fn := range module.Functions {
		if fn.Line > 0 {
			enabled = true
			break
		}
	}
	if !enabled {
		return &debugMetadataState{}
	}
	sourcePath := module.SourcePath
	if sourcePath == "" {
		sourcePath = "<stdin>"
	}
	fileName := filepath.Base(sourcePath)
	directory := filepath.Dir(sourcePath)
	if fileName == "." || fileName == "" {
		fileName = sourcePath
	}
	if directory == "." || directory == "" {
		directory = "."
	}
	state := &debugMetadataState{
		enabled:          true,
		fileName:         fileName,
		directory:        directory,
		compileUnitID:    0,
		fileID:           4,
		subroutineTypeID: 5,
		functionIDs:      map[string]int{},
		locationIDs:      map[string]int{},
	}
	nextID := 6
	for _, fn := range module.Functions {
		state.functionIDs[fn.Name] = nextID
		nextID++
	}
	for _, fn := range module.Functions {
		state.locationIDs[fn.Name] = nextID
		nextID++
	}
	return state
}

func emitDebugMetadata(buf *bytes.Buffer, module *ir.Module, debugState *debugMetadataState) {
	if debugState == nil || !debugState.enabled {
		return
	}
	buf.WriteString("\n")
	fmt.Fprintf(buf, "!llvm.dbg.cu = !{!%d}\n", debugState.compileUnitID)
	buf.WriteString("!llvm.module.flags = !{!1, !2}\n")
	buf.WriteString("!llvm.ident = !{!3}\n")
	fmt.Fprintf(buf, "!%d = distinct !DICompileUnit(language: DW_LANG_C99, file: !%d, producer: \"jayess\", isOptimized: false, runtimeVersion: 0, emissionKind: FullDebug)\n", debugState.compileUnitID, debugState.fileID)
	buf.WriteString("!1 = !{i32 2, !\"Debug Info Version\", i32 3}\n")
	buf.WriteString("!2 = !{i32 7, !\"Dwarf Version\", i32 4}\n")
	buf.WriteString("!3 = !{!\"jayess\"}\n")
	fmt.Fprintf(buf, "!%d = !DIFile(filename: \"%s\", directory: \"%s\")\n", debugState.fileID, escapeLLVMMetadataString(debugState.fileName), escapeLLVMMetadataString(debugState.directory))
	fmt.Fprintf(buf, "!%d = !DISubroutineType(types: !{})\n", debugState.subroutineTypeID)
	for _, fn := range module.Functions {
		line := fn.Line
		if line <= 0 {
			line = 1
		}
		linkageName := emittedFunctionName(fn.Name)
		fmt.Fprintf(buf, "!%d = distinct !DISubprogram(name: \"%s\", linkageName: \"%s\", scope: !%d, file: !%d, line: %d, type: !%d, scopeLine: %d, spFlags: DISPFlagDefinition, unit: !%d)\n",
			debugState.functionIDs[fn.Name],
			escapeLLVMMetadataString(fn.Name),
			escapeLLVMMetadataString(linkageName),
			debugState.fileID,
			debugState.fileID,
			line,
			debugState.subroutineTypeID,
			line,
			debugState.compileUnitID,
		)
		column := fn.Column
		if column <= 0 {
			column = 1
		}
		fmt.Fprintf(buf, "!%d = !DILocation(line: %d, column: %d, scope: !%d)\n",
			debugState.locationIDs[fn.Name],
			line,
			column,
			debugState.functionIDs[fn.Name],
		)
	}
}

func debugLocationIDForFunctionHeader(line string, module *ir.Module, debugState *debugMetadataState) int {
	if debugState == nil || !debugState.enabled {
		return 0
	}
	for _, fn := range module.Functions {
		emittedName := emittedFunctionName(fn.Name)
		if strings.Contains(line, "@"+emittedName+"(") {
			return debugState.locationIDs[fn.Name]
		}
	}
	return 0
}

func applyFunctionDebugLocations(irText string, module *ir.Module, debugState *debugMetadataState) string {
	if debugState == nil || !debugState.enabled {
		return irText
	}
	lines := strings.SplitAfter(irText, "\n")
	currentLocationID := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "define ") {
			currentLocationID = debugLocationIDForFunctionHeader(line, module, debugState)
			continue
		}
		if trimmed == "}" {
			currentLocationID = 0
			continue
		}
		if currentLocationID == 0 {
			continue
		}
		if !strings.HasPrefix(line, "  ") || !strings.Contains(line, " call ") || strings.Contains(line, "!dbg !") {
			continue
		}
		lines[i] = strings.TrimRight(line, "\n") + fmt.Sprintf(", !dbg !%d\n", currentLocationID)
	}
	return strings.Join(lines, "")
}
