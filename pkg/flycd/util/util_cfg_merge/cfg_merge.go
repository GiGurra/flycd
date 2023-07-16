package util_cfg_merge

import (
	"fmt"
	"github.com/samber/lo"
	"reflect"
)

type MergeConfig struct {
	SliceStrategy  SliceStrategy
	SliceMergeKeys []string
}

type SliceStrategy string

const (
	SliceStrategyAppendNoDuplicates      SliceStrategy = "append-no-duplicates"       // default: append the new slice to the original slice, but ignore duplicates
	SliceStrategyAppend                  SliceStrategy = "append"                     // append the new slice to the original slice
	SliceStrategyTruncateAndReplace      SliceStrategy = "replace"                    // replace the original slice with the new slice
	SliceStrategyMergeOverlapReplaceRest SliceStrategy = "merge-overlap-replace-rest" // replace the original slice with the new slice
)

// MergeMaps This only works with basic builtin types - NOT with structs or pointers inside the maps
func MergeMaps(
	base map[string]any,
	overlay map[string]any,
	configs ...MergeConfig,
) map[string]any {

	if overlay == nil {
		return base
	}

	if base == nil {
		return overlay
	}

	config := MergeConfig{
		SliceStrategy: SliceStrategyAppendNoDuplicates,
	}
	if len(configs) > 0 {
		config = configs[0]
	}

	merged := doMerge(overlay, base, config)

	result, ok := merged.(map[string]any)
	if !ok {
		panic("Should never be possible: failed to cast merged map to map[string]any")
	}

	return result
}

// Below is copy pasta from https://github.com/ieee0824/go-deepmerge,
// which is also under MIT license.
// + one nil fix + no longer returning errors

func doMerge(
	overlay any,
	base any,
	config MergeConfig,
) any {

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

		// Make an explicit new map to avoid modifying inputs
		resultMap := map[string]any{}

		for k, baseVal := range convertMap(base) {
			resultMap[k] = baseVal
		}

		for k, overlayVal := range convertMap(overlay) {
			baseVal, existsInBase := resultMap[k]
			if !existsInBase {
				resultMap[k] = overlayVal
			} else {
				mergedVal := doMerge(overlayVal, baseVal, config)
				resultMap[k] = mergedVal
			}
		}

		return resultMap

	case reflect.Slice:

		switch config.SliceStrategy {
		case SliceStrategyAppendNoDuplicates:
			resultSlice := convertSlice(base)
			for _, overlayItem := range convertSlice(overlay) {
				if !lo.Contains(resultSlice, overlayItem) {
					resultSlice = append(resultSlice, overlayItem)
				}
			}
			return resultSlice
		case SliceStrategyAppend:
			return append(convertSlice(base), convertSlice(overlay)...)
		case SliceStrategyTruncateAndReplace:
			return overlay
		case SliceStrategyMergeOverlapReplaceRest:

			resultSlice := []any{}
			baseSlice := convertSlice(base)
			baseSliceMaps := lo.Map(lo.Filter(baseSlice, func(item any, index int) bool {
				return item != nil && reflect.TypeOf(item).Kind() == reflect.Map
			}), func(item any, _ int) map[string]any {
				return convertMap(item)
			})

			for _, overlayItem := range convertSlice(overlay) {

				// Check if the value is already in the slice
				// The equality test is performed if both baseVal and overlayItem are of type map or object
				// The equality is given if all the primitive

				if overlayItem == nil {
					resultSlice = append(resultSlice, overlayItem)
					continue
				}

				if reflect.TypeOf(overlayItem).Kind() != reflect.Map {
					resultSlice = append(resultSlice, overlayItem)
					continue
				}

				overlayItem := convertMap(overlayItem)

				// We need to check if we are to merge this with base array element or not
				matched := lo.Filter(baseSliceMaps, func(baseItem map[string]any, index int) bool {

					baseItemKeys := lo.Keys(baseItem)
					overlayItemKeys := lo.Keys(overlayItem)

					// Find same keys overlapping with SliceMergeKeys
					sameKeys := lo.Filter(baseItemKeys, func(key string, index int) bool {
						return lo.Contains(config.SliceMergeKeys, key) && lo.IndexOf(overlayItemKeys, key) != -1
					})

					// only allow same keys if the values they point to are of the same type and of primitive (int, string, bool, float)
					sameKeys = lo.Filter(sameKeys, func(key string, index int) bool {
						baseValue := baseItem[key]
						overlayValue := overlayItem[key]
						if baseValue == nil || overlayValue == nil {
							return baseValue == nil && overlayValue == nil
						}
						baseKind := reflect.TypeOf(baseValue).Kind()
						overlayKind := reflect.TypeOf(overlayValue).Kind()
						if baseKind != overlayKind {
							return false
						}
						return lo.Contains(mergeKeyKinds, baseKind)
					})

					if len(sameKeys) == 0 {
						return false
					}

					// Check if all the values are the same
					for _, key := range sameKeys {
						baseValue := baseItem[key]
						overlayValue := overlayItem[key]
						if baseValue != overlayValue {
							return false
						}
					}

					return true
				})

				if len(matched) == 0 {
					resultSlice = append(resultSlice, overlayItem)
					continue
				}

				// Merge the values
				valuesToAppend := lo.Map(matched, func(baseItem map[string]any, index int) any {
					result := doMerge(overlayItem, baseItem, config)
					return result
				})

				resultSlice = append(resultSlice, valuesToAppend...)
			}

			return resultSlice
		default:
			return append(convertSlice(base), convertSlice(overlay)...)
		}
	default:
		return overlay
	}
}

var mergeKeyKinds = []reflect.Kind{
	reflect.Bool,
	reflect.String,
	// int types
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
	// float types
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Float32,
	reflect.Float64,
	reflect.Complex64,
	reflect.Complex128,
}

// Because go has strongly typed (non-erasure) generics,
// we can't just cast to map[string]any. Instead, we have to
// explicitly convert the map to a map[string]any (unless it already is one).
func convertMap(i any) map[string]any {
	rightMap, ok := i.(map[string]any)
	if ok {
		return rightMap
	} else {

		result := map[string]any{}

		v := reflect.ValueOf(i)
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			result[key.String()] = value.Interface()
		}
		return result
	}
}

func convertSlice(i any) []any {
	ret := []any{}

	switch i.(type) {
	case []any:
		return i.([]any)
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
