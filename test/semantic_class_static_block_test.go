package test

import "testing"

func TestSemanticAnalyzesClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#seed = 1;
			static {
				this.value = this.#seed;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsReturnInClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static {
				return 1;
			}
		}
	`)
	requireSemanticError(t, err, "return outside function")
}

func TestSemanticAllowsSuperInDerivedClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Base {
			static value() {
				return 1;
			}
		}
		class Counter extends Base {
			static {
				this.value = super.value();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsSuperInBaseClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static {
				super.value();
			}
		}
	`)
	requireSemanticError(t, err, "super in class without extends")
}

func TestSemanticRejectsUnknownPrivateAccessInClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static {
				this.#missing;
			}
		}
	`)
	requireSemanticError(t, err, "private member #missing is not declared")
}
