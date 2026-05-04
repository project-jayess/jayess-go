package test

import "testing"

func TestSemanticAllowsDeclaredSuperclass(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {}
		class Counter extends BaseCounter {
			constructor(value) {
				super(value);
			}
			total() {
				return super.total();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesMethodOverrideShape(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {
			total() {
				return 1;
			}
		}
		class Counter extends BaseCounter {
			total() {
				return super.total() + 1;
			}
		}
		const counter = new Counter();
		counter.total();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesPrototypeChainMethodCallShape(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {
			total() {
				return 1;
			}
		}
		class Counter extends BaseCounter {
			total() {
				return BaseCounter.prototype.total.call(this) + 1;
			}
		}
		const counter = new Counter();
		counter.total();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUndeclaredSuperclass(t *testing.T) {
	err := analyzeSource(t, `
		class Counter extends MissingBase {}
	`)
	requireSemanticError(t, err, "use of MissingBase before declaration")
}

func TestSemanticRejectsNonConstructableSuperclass(t *testing.T) {
	err := analyzeSource(t, `
		const BaseCounter = 1;
		class Counter extends BaseCounter {}
	`)
	requireSemanticError(t, err, "extends target BaseCounter is not constructable")
}

func TestSemanticAllowsImportedSuperclass(t *testing.T) {
	err := analyzeSource(t, `
		import { BaseCounter } from "./base.js";
		class Counter extends BaseCounter {}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsSuperOutsideClassMethod(t *testing.T) {
	err := analyzeSource(t, `super.value;`)
	requireSemanticError(t, err, "super outside class method")
}

func TestSemanticRejectsSuperInClassWithoutExtends(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			total() {
				return super.total();
			}
		}
	`)
	requireSemanticError(t, err, "super in class without extends")
}

func TestSemanticRejectsSuperInClassFieldInitializer(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {}
		class Counter extends BaseCounter {
			total = super.total();
		}
	`)
	requireSemanticError(t, err, "super outside class method")
}

func TestSemanticRejectsSuperCallOutsideConstructor(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {}
		class Counter extends BaseCounter {
			reset() {
				super();
			}
		}
	`)
	requireSemanticError(t, err, "super call outside constructor")
}

func TestSemanticRejectsSuperCallInStaticBlock(t *testing.T) {
	err := analyzeSource(t, `
		class BaseCounter {}
		class Counter extends BaseCounter {
			static {
				super();
			}
		}
	`)
	requireSemanticError(t, err, "super call outside constructor")
}
