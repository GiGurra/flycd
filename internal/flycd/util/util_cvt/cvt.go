package util_cvt

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

func MapToStruct[T any](m map[string]any) (T, error) {
	var result T
	err := mapstructure.Decode(m, &result)
	if err != nil {
		return result, fmt.Errorf("failed to decode map to struct: %w", err)
	}
	return result, nil
}

func StructToMap[T any](s T) (map[string]any, error) {
	var result map[string]any
	err := mapstructure.Decode(s, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode struct to map: %w", err)
	}
	return result, nil
}
