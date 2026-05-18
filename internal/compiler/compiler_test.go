package compiler

import "testing"

func TestResolveProtoFileArg_UsesFirstMatchingImportPath(t *testing.T) {
	got := ResolveProtoFileArg("araviec/sys/v1/cods.proto", []string{".", "araviec"})
	want := "araviec/sys/v1/cods.proto"
	if got != want {
		t.Fatalf("ResolveProtoFileArg() = %q, want %q", got, want)
	}
}

func TestResolveProtoFileArg_UsesProtoRootWhenItMatchesFirst(t *testing.T) {
	got := ResolveProtoFileArg("protos/sys/v1/cods.proto", []string{"protos", "."})
	want := "sys/v1/cods.proto"
	if got != want {
		t.Fatalf("ResolveProtoFileArg() = %q, want %q", got, want)
	}
}

func TestResolveProtoFileArg_ReturnsUnchangedWithoutMatch(t *testing.T) {
	got := ResolveProtoFileArg("sys/v1/cods.proto", []string{"third_party"})
	want := "sys/v1/cods.proto"
	if got != want {
		t.Fatalf("ResolveProtoFileArg() = %q, want %q", got, want)
	}
}
