package util_cvt

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

func MapYamlToStruct[T any](m map[string]any) (T, error) {
	var result T
	err := structDecode(m, &result)
	if err != nil {
		return result, fmt.Errorf("failed to decode map to struct: %w", err)
	}
	return result, nil
}

// Decode takes an input structure and uses reflection to translate it to
// the output structure. output must be a pointer to a map or struct.
func structDecode(mp interface{}, strct interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           strct,
		WeaklyTypedInput: true,
		TagName:          "yaml",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(mp)
}
