package util_toml

import (
	"bytes"
	"github.com/BurntSushi/toml"
)

func Marshal(x any) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	encoder := toml.NewEncoder(buf)
	err := encoder.Encode(x)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Unmarshal(x string, v any) error {
	_, err := toml.Decode(x, v)
	return err
}
