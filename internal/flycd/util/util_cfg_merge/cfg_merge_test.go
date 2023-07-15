package util_cfg_merge

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestMerge_TopLevel(t *testing.T) {
	base := map[string]any{
		"foo": "bar",
		"bar": "bar",
	}
	overlay := map[string]any{
		"foo": "baz",
	}
	expected := map[string]any{
		"foo": "baz",
		"bar": "bar",
	}
	actual, err := Merge(base, overlay)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}
