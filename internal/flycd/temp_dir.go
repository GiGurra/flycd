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
