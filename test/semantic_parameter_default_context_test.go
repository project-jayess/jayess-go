package test

import "testing"

func TestSemanticAllowsThisInClassMethodParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			value = 1;
			add(next = this.value) {
				return next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInObjectMethodParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			value: 1,
			add(next = this.value) {
				return next;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInClassMethodDestructuringParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			value = 1;
			add({ next = this.value }) {
				return next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInObjectMethodDestructuringParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			value: 1,
			add({ next = this.value }) {
				return next;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInFunctionParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		function add(next = this.value) {
			return next;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInFunctionDestructuringParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		function add({ next = this.value }) {
			return next;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
