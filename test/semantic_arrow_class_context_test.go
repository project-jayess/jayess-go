package test

import "testing"

func TestSemanticAllowsArrowPrivateAccessInClassFieldInitializer(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#seed = 1;
			value = () => this.#seed;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArrowSuperInDerivedClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Base {
			static value() {
				return 1;
			}
		}
		class Counter extends Base {
			static {
				const read = () => super.value();
				this.value = read();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsArrowSuperInBaseClassStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static {
				const read = () => super.value();
				read();
			}
		}
	`)
	requireSemanticError(t, err, "super in class without extends")
}
