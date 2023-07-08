package util_work_dir

import (
	"flycd/internal/flycd/util/util_cmd"
	"fmt"
	"os"
)

type WorkDir struct {
	Root string
	Cwd  string
}

func (t *WorkDir) Remove() {
	err := os.RemoveAll(t.Root)
	if err != nil {
		fmt.Printf("error removing temp dir %s: %s", t.Root, err)
	}
}

func NewTempDir() (WorkDir, error) {
	tempDir, err := os.MkdirTemp("", "flycd")
	if err != nil {
		return WorkDir{}, fmt.Errorf("error creating temp tempDir: %w", err)
	}
	return WorkDir{
		Root: tempDir,
		Cwd:  tempDir,
	}, nil
}

func NewWorkDir(path string) WorkDir {
	return WorkDir{
		Root: path,
		Cwd:  path,
	}
}

func CwDir() (WorkDir, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return WorkDir{}, fmt.Errorf("error getting current working directory: %w", err)
	}
	return WorkDir{
		Root: cwd,
		Cwd:  cwd,
	}, nil
}

func (t *WorkDir) RunCommand(command string, args ...string) (string, error) {
	return util_cmd.Run(t.Cwd, command, args...)
}

func (t *WorkDir) ReadFile(name string) (string, error) {
	data, err := os.ReadFile(t.Cwd + "/" + name)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", name, err)
	}
	return string(data), nil
}

func (t *WorkDir) WriteFile(name string, contents string) error {
	return os.WriteFile(t.Cwd+"/"+name, []byte(contents), 0644)
}
