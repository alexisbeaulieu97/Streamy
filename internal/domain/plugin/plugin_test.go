package plugin

import "testing"

func TestIsSupportedType(t *testing.T) {
	if !IsSupportedType(TypeCommand) {
		t.Fatal("expected command type to be supported")
	}
	if IsSupportedType(Type("other")) {
		t.Fatal("did not expect other type to be supported")
	}
}
