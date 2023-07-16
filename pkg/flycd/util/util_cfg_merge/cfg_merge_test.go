package util_cfg_merge

import (
	"fmt"
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

	actual := MergeMaps(base, overlay)
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

	actual := MergeMaps(base, overlay)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}

func TestMerge_NilNil(t *testing.T) {
	var base map[string]any = nil
	var overlay map[string]any = nil
	var expected map[string]any = nil

	actual := MergeMaps(base, overlay)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}

func TestMerge_mapInMap(t *testing.T) {

	base := map[string]any{
		"common1": "base-common1",
		"common2": "base-common2",
		"unique1": "base-unique1",
		"blah":    "hah",
		"subMapKey": map[string]any{
			"sub1": "base-sub1",
			"sub2": "base-sub2",
		},
	}

	overlay := map[string]any{
		"common1": "base-common1",
		"common2": "base-common2x",
		"uq":      "vq",
		"subMapKey": map[string]any{
			"sub1": "overlay-sub1",
		},
	}

	expected := map[string]any{
		"common1": "base-common1",
		"common2": "base-common2x",
		"uq":      "vq",
		"unique1": "base-unique1",
		"blah":    "hah",
		"subMapKey": map[string]any{
			"sub1": "overlay-sub1",
			"sub2": "base-sub2",
		},
	}

	actual := MergeMaps(base, overlay)

	fmt.Printf("  actual: %v\n", actual)
	fmt.Printf("expected: %v\n", expected)

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}

func TestMerge_Arrays(t *testing.T) {

	// Case:
	// - the base array contains objects/maps
	// - the overlay array contains objects/maps
	// -> merge deep, calculate keys using common primitive fields (non-object, non-map, non-slice)

	baseArray := []any{
		map[string]any{
			"common1": "base-common1",
			"common2": "base-common2",
			"unique1": "base-unique1",
			"subMapKey": map[string]any{
				"sub1": "base-sub1",
				"sub2": "base-sub2",
			},
		},
		map[string]any{
			"xcommon1": "base-common1",
			"xcommon2": "base-common2",
			"unique1":  "base-unique1",
			"subMapKey": map[string]any{
				"sub1": "base-sub1",
				"sub2": "base-sub2",
			},
		},
	}

	overlayArray := []any{
		map[string]any{
			"common1": "base-common1",
			"common2": "base-common2",
			"subMapKey": map[string]any{
				"sub1": "overlay-sub1",
			},
		},
	}

	expectedArray := []any{
		map[string]any{
			"common1": "base-common1",
			"common2": "base-common2",
			"unique1": "base-unique1",
			"subMapKey": map[string]any{
				"sub1": "overlay-sub1",
				"sub2": "base-sub2",
			},
		},
	}

	base := map[string]any{
		"foo": baseArray,
	}

	overlay := map[string]any{
		"foo": overlayArray,
	}

	expected := map[string]any{
		"foo": expectedArray,
	}

	actual := MergeMaps(base, overlay, "common1", "common2")
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("Expected %v, diff: %s", expected, diff)
	}
}
