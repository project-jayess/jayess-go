package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeUtilCapabilities(t *testing.T) {
	for _, name := range []string{"format", "inspect"} {
		if !jayessruntime.HasUtilCapability(name) {
			t.Fatalf("expected util capability %s", name)
		}
	}
}

func TestRuntimeUtilFormatAndInspect(t *testing.T) {
	if got := jayessruntime.UtilFormat("hello %s", "jayess"); got != "hello jayess" {
		t.Fatalf("unexpected formatted string %q", got)
	}
	label := "value"
	if got := jayessruntime.UtilFormat(label, 42, "ok"); got != "value 42 \"ok\"" {
		t.Fatalf("unexpected joined format %q", got)
	}
	buffer := &jayessruntime.Buffer{Data: []byte{0x01, 0x02}}
	if got := jayessruntime.UtilInspect(buffer); got != "<Buffer 0102>" {
		t.Fatalf("unexpected buffer inspect %q", got)
	}
	if got := jayessruntime.UtilInspect(map[string]any{"b": 2, "a": "one"}); got != `{ a: "one", b: 2 }` {
		t.Fatalf("unexpected map inspect %q", got)
	}
}
