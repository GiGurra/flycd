package util_cfg_merge

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestMerge_TopLevel(t *testing.T) {
	base := map[string]any{
		"foo": "bar",
		"bar": "bar",
		"bo":  nil,
		"yo2": nil,
	}
	overlay := map[string]any{
		"foo": "baz",
		"yo":  "ho",
		"yo2": "ho",
	}
	expected := map[string]any{
		"foo": "baz",
		"bar": "bar",
		"yo":  "ho",
		"yo2": "ho",
		"bo":  nil,
	}

	actual := Merge(base, overlay)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}

func TestMerge_Deep(t *testing.T) {
	base := map[string]any{
		"foo": map[string]any{
			"bar": "bar",
			"baz": "baz",
			"qux": []any{"foo", "bar", "baz"},
		},
	}
	overlay := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
			"qux": []any{"foo2", "bar2", "baz2"},
		},
	}
	expected := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
			"baz": "baz",
			"qux": []any{"foo2", "bar2", "baz2"},
		},
	}

	actual := Merge(base, overlay)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}
