package util_cfg_merge

import (
	"fmt"
	"reflect"
)

func Merge(base map[string]any, overlay map[string]any) map[string]any {

	if overlay == nil {
		return base
	}

	if base == nil {
		return overlay
	}

	merged := doMerge(overlay, base)

	result, ok := merged.(map[string]any)
	if !ok {
		panic("Should never be possible: failed to cast merged map to map[string]any")
	}

	return result
}

// Below is copy pasta from https://github.com/ieee0824/go-deepmerge,
// which is also under MIT license.
// + one nil fix + no longer returning errors

func doMerge(overlay, base interface{}) interface{} {

	if overlay == nil {
		return base
	}

	if base == nil {
		return overlay
	}

	overlayType := reflect.TypeOf(overlay)
	baseType := reflect.TypeOf(base)
	if overlayType.Kind() != baseType.Kind() {
		fmt.Printf(
			"Returning overlay data without merge. "+
				"Type mismatch in merge. overlay type: %v, base type: %v."+
				"overlay data: %v, base data: %v", overlayType.Kind(), baseType.Kind(), overlay, base,
		)
		return overlay
	}

	switch overlayType.Kind() {
	case reflect.Map:
		srcMap := overlay.(map[string]interface{})
		for k, dstVal := range base.(map[string]interface{}) {
			srcVal, ok := srcMap[k]
			if !ok {
				srcMap[k] = dstVal
			} else {
				mergedVal := doMerge(srcVal, dstVal)
				srcMap[k] = mergedVal
			}
		}
		return overlay
	case reflect.Slice:

		// Here is how we differ from the go-deepmerge package: We do not merge arrays
		// but instead replace them.

		// In the future we might support intelligent merging of arrays (kustomize style),
		// but for now we just replace them.

		////return append(overlay.([]interface{}), base.([]interface{})...), nil
		//srcSlice := convertSlice(overlay)
		//dstSlice := convertSlice(base)
		//return append(srcSlice, dstSlice...), nil

		return overlay
	default:
		return overlay
	}
}

func convertSlice(i interface{}) []interface{} {
	ret := []interface{}{}

	switch i.(type) {
	case []interface{}:
		return i.([]interface{})
	case []string:
		for _, v := range i.([]string) {
			ret = append(ret, v)
		}
		return ret
	case []int:
		for _, v := range i.([]int) {
			ret = append(ret, v)
		}
		return ret
	case []float64:
		for _, v := range i.([]float64) {
			ret = append(ret, v)
		}
		return ret
	case []float32:
		for _, v := range i.([]float32) {
			ret = append(ret, v)
		}
		return ret
	case []byte:
		return append(ret, i)
	}
	return nil
}
