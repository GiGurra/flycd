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

func (t *WorkDir) RemoveAll() {
	err := os.RemoveAll(t.Root)
	if err != nil {
		fmt.Printf("error removing dir %s: %s", t.Root, err)
	}
}

func NewTempDir(name string, root string) (WorkDir, error) {
	pattern := "flycd"
	if name != "" {
		pattern = name
	}
	tempDir, err := os.MkdirTemp(root, pattern)
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

func (t *WorkDir) NewCommand(command string, args ...string) util_cmd.Command {
	return util_cmd.NewCommandA(command, args...).WithCwd(t.Cwd)
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
