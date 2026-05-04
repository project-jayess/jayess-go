package tooling

import (
	"path/filepath"

	"jayess-go/binding"
	"jayess-go/llvmbackend"
)

const bindingObjectWorkDir = "temp/jayess-bindings"

func bindingObjectFiles(plan binding.BuildPlan, target llvmbackend.TargetConfig) []string {
	objects := make([]string, 0, len(plan.CompileUnits))
	ext := bindingObjectExtension(target)
	for index, unit := range plan.CompileUnits {
		objects = append(objects, filepath.Join(bindingObjectWorkDir, bindingObjectBase(index, unit)+ext))
	}
	return objects
}

func bindingObjectBase(index int, unit binding.CompileUnit) string {
	module := sanitizeObjectName(filepath.Base(unit.ModulePath))
	source := sanitizeObjectName(filepath.Base(unit.Source))
	if module == "" {
		module = "binding"
	}
	if source == "" {
		source = "source"
	}
	return itoa(index) + "-" + module + "-" + source
}

func sanitizeObjectName(value string) string {
	ext := filepath.Ext(value)
	value = value[:len(value)-len(ext)]
	out := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '_' || character == '-' {
			out = append(out, character)
			continue
		}
		out = append(out, '_')
	}
	return string(out)
}

func bindingObjectExtension(target llvmbackend.TargetConfig) string {
	if target.Name == "windows-x64" {
		return ".obj"
	}
	return ".o"
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
