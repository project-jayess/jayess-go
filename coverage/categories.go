package coverage

type Category struct {
	Name      string
	TestScope string
}

func Categories() []Category {
	return []Category{
		{Name: "lexer", TestScope: "./lexer"},
		{Name: "parser", TestScope: "./test/parser_*"},
		{Name: "ast", TestScope: "./test/ast_*"},
		{Name: "semantic", TestScope: "./test/semantic_*"},
		{Name: "type-checking", TestScope: "./test/typesys_*"},
		{Name: "lifetime-escape", TestScope: "./test/lifetime_* ./test/escape_*"},
		{Name: "codegen", TestScope: "./test/codegen_*"},
		{Name: "llvm-ir", TestScope: "./test/llvm_*"},
		{Name: "runtime", TestScope: "./test/runtime_*"},
		{Name: "filesystem", TestScope: "./test/runtime_filesystem_test.go"},
		{Name: "network", TestScope: "./test/runtime_http_* ./test/runtime_tcp_* ./test/runtime_udp_*"},
		{Name: "module-resolution", TestScope: "./test/resolver_*"},
		{Name: "cross-platform", TestScope: "./test/target_cross_platform_test.go"},
		{Name: "e2e-native", TestScope: "./test/e2e_*"},
		{Name: "regression", TestScope: "./test/*_test.go"},
	}
}

func HasCategory(name string) bool {
	for _, category := range Categories() {
		if category.Name == name {
			return category.TestScope != ""
		}
	}
	return false
}
