package test

import "testing"

func TestSemanticRejectsDuplicateConstructor(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			constructor() {}
			constructor(value) {
				this.value = value;
			}
		}
	`)
	requireSemanticError(t, err, "duplicate constructor")
}

func TestSemanticRejectsDuplicatePrivateField(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			#value = 2;
		}
	`)
	requireSemanticError(t, err, "duplicate private member #value")
}

func TestSemanticRejectsDuplicatePrivateMethodAndField(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			#value() {
				return 2;
			}
		}
	`)
	requireSemanticError(t, err, "duplicate private member #value")
}

func TestSemanticAllowsPrivateGetterSetterPair(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			get #value() {
				return 1;
			}
			set #value(next) {
				this.#value = next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsStaticPrototypeMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static prototype() {}
		}
	`)
	requireSemanticError(t, err, "static class member cannot be named prototype")
}

func TestSemanticRejectsStaticPrototypeField(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static prototype = 1;
		}
	`)
	requireSemanticError(t, err, "static class member cannot be named prototype")
}

func TestSemanticRejectsStaticPrototypeAccessor(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			static get prototype() {
				return 1;
			}
		}
	`)
	requireSemanticError(t, err, "static class member cannot be named prototype")
}

func TestSemanticAllowsInstancePrototypeMember(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			prototype() {
				return 1;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
