package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeURLCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"parse",
		"format",
		"parseQuery",
		"stringifyQuery",
		"encode",
		"decode",
		"fileURLToPath",
		"pathToFileURL",
	}
	for _, name := range expected {
		if !jayessruntime.HasURLCapability(name) {
			t.Fatalf("expected URL runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsURLUtilitySurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(input, filePath) {
			const parsed = url.parse(input);
			const formatted = url.format(parsed);
			const query = url.parseQuery("a=1&b=two");
			const encodedQuery = url.stringifyQuery(query);
			const escaped = url.encode(formatted);
			const unescaped = url.decode(escaped);
			const fileURL = url.pathToFileURL(filePath);
			const localPath = url.fileURLToPath(fileURL);
			return encodedQuery || unescaped || localPath;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeURLCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.URLCapabilities() {
		if capability.Name == "" {
			t.Fatalf("URL capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("URL capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("URL capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestRuntimeURLParseFormatAndQuery(t *testing.T) {
	parsed, err := jayessruntime.URLParse("https://example.com/a?b=two#frag")
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "example.com" || parsed.Query["b"][0] != "two" {
		t.Fatalf("unexpected parsed URL: %#v", parsed)
	}
	formatted := jayessruntime.URLFormat(parsed)
	if formatted != "https://example.com/a?b=two#frag" {
		t.Fatalf("unexpected formatted URL %q", formatted)
	}
	query, err := jayessruntime.URLParseQuery("a=1&b=two")
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := jayessruntime.URLStringifyQuery(query); got != "a=1&b=two" {
		t.Fatalf("unexpected query string %q", got)
	}
	encoded := jayessruntime.URLEncode("a b")
	decoded, err := jayessruntime.URLDecode(encoded)
	if err != nil || decoded != "a b" {
		t.Fatalf("decode got %q err=%v", decoded, err)
	}
}

func TestRuntimeURLFilePathRoundTrip(t *testing.T) {
	fileURL := jayessruntime.URLPathToFileURL("tmp/jayess.txt")
	path, err := jayessruntime.URLFileURLToPath(fileURL)
	if err != nil {
		t.Fatalf("file url to path: %v", err)
	}
	if path == "" {
		t.Fatal("expected file path")
	}
}

func TestSemanticRejectsTopLevelURLRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var url = {};`)
	requireSemanticError(t, err, "duplicate declaration url")
}
