package llvmbackend

type ToolchainStep string

const (
	LLVMVerifyStep   ToolchainStep = "llvm-verify"
	OptStep          ToolchainStep = "opt"
	LLCStep          ToolchainStep = "llc"
	ClangCompileStep ToolchainStep = "clang-compile"
	ClangLinkStep    ToolchainStep = "clang-link"
)

type ToolchainInterop struct {
	Steps       []ToolchainStep
	StableABI   bool
	Diagnostics []string
}

func DefaultToolchainInterop() ToolchainInterop {
	return ToolchainInterop{
		Steps: []ToolchainStep{
			LLVMVerifyStep,
			OptStep,
			LLCStep,
			ClangCompileStep,
			ClangLinkStep,
		},
		StableABI:   true,
		Diagnostics: []string{"missing LLVM tool", "linker failure", "ABI mismatch"},
	}
}
