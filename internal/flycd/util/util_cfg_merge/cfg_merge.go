package util_cfg_merge

import (
	"errors"
	"fmt"
	"reflect"
)

func Merge(base map[string]any, overlay map[string]any) (map[string]any, error) {
	merged, err := doMerge(overlay, base)
	if err != nil {
		return nil, fmt.Errorf("failed to merge overlay: %w", err)
	}

	result, ok := merged.(map[string]any)
	if !ok {
		return nil, errors.New("failed to cast merged map to map[string]any")
	}

	return result, nil
}

// Below is copy pasta from https://github.com/ieee0824/go-deepmerge,
// which is also under MIT license.

var (
	TypeNotMatchErr = errors.New("type not match")
)

func doMerge(src, dst interface{}) (interface{}, error) {
	srcType := reflect.TypeOf(src)
	dstType := reflect.TypeOf(dst)
	if srcType.Kind() != dstType.Kind() {
		return nil, TypeNotMatchErr
	}

	switch srcType.Kind() {
	case reflect.Map:
		srcMap := src.(map[string]interface{})
		for k, dstVal := range dst.(map[string]interface{}) {
			srcVal, ok := srcMap[k]
			if !ok {
				srcMap[k] = dstVal
			} else {
				mergedVal, err := doMerge(srcVal, dstVal)
				if err != nil {
					return nil, err
				}
				srcMap[k] = mergedVal
			}
		}
		return src, nil
	case reflect.Slice:

		// Here is how we differ from the go-deepmerge package: We do not merge arrays
		// but instead replace them.

		// In the future we might support intelligent merging of arrays (kustomize style),
		// but for now we just replace them.

		////return append(src.([]interface{}), dst.([]interface{})...), nil
		//srcSlice := convertSlice(src)
		//dstSlice := convertSlice(dst)
		//return append(srcSlice, dstSlice...), nil

		return src, nil
	default:
		return src, nil
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
