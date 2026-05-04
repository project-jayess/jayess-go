package test

import "testing"

func TestSemanticAnalyzesPrivateClassMembers(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			#next() {
				return this.#value;
			}
			get #current() {
				return this.#value;
			}
			set #current(next) {
				this.#value = next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesPrivateAsyncClassMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Loader {
			async #load() {
				await 1;
			}
			run() {
				return this.#load();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesPrivateGeneratorClassMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Ids {
			*#values() {
				yield 1;
			}
			run() {
				return this.#values();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownPrivateMemberAccess(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			value() {
				return this.#missing;
			}
		}
	`)
	requireSemanticError(t, err, "private member #missing is not declared")
}

func TestSemanticAnalyzesPrivateMemberAssignment(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			setValue(next) {
				this.#value = next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownPrivateMemberAssignment(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			setValue(next) {
				this.#missing = next;
			}
		}
	`)
	requireSemanticError(t, err, "private member #missing is not declared")
}

func TestSemanticRejectsPrivateMemberAccessOutsideClassMethod(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {};
		counter.#value;
	`)
	requireSemanticError(t, err, "private member access outside class method")
}

func TestSemanticRejectsOtherClassPrivateMemberAccess(t *testing.T) {
	err := analyzeSource(t, `
		class Other {
			#secret = 1;
		}
		class Counter {
			value(other) {
				return other.#secret;
			}
		}
	`)
	requireSemanticError(t, err, "private member #secret is not declared")
}
