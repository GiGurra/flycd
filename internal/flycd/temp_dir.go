package flycd

import (
	"fmt"
	"os"
)

type TempDir struct {
	Root string
	Cwd  string
}

func (t *TempDir) Remove() {
	err := os.RemoveAll(t.Root)
	if err != nil {
		fmt.Printf("error removing temp dir %s: %s", t.Root, err)
	}
}

func NewTempDir() (TempDir, error) {
	tempDir, err := os.MkdirTemp("", "flycd")
	if err != nil {
		return TempDir{}, fmt.Errorf("error creating temp tempDir: %w", err)
	}
	return TempDir{
		Root: tempDir,
		Cwd:  tempDir,
	}, nil
}

func (t *TempDir) RunCommand(command string, args ...string) (string, error) {
	return runCommand(t.Cwd, command, args...)
}

func (t *TempDir) ReadFile(name string) (string, error) {
	data, err := os.ReadFile(t.Cwd + "/" + name)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", name, err)
	}
	return string(data), nil
}

func (t *TempDir) WriteFile(name string, contents string) error {
	return os.WriteFile(t.Cwd+"/"+name, []byte(contents), 0644)
}
