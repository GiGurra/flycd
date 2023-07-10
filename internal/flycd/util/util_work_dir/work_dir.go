package util_work_dir

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
	"os"
)

type WorkDirImpl struct {
	root string
	cwd  string
}

type WorkDir interface {
	NewCommand(command string, args ...string) util_cmd.Command
	ReadFile(name string) (string, error)
	WriteFile(name string, content string) error
	Root() string
	Cwd() string
	SetCwd(path string)
}

type TempDirImpl struct {
	WorkDirImpl
}

type TempDir interface {
	WorkDir
	RemoveAll()
}

func (t TempDirImpl) RemoveAll() {
	err := os.RemoveAll(t.Root())
	if err != nil {
		fmt.Printf("error removing dir %s: %s", t.Root(), err)
	}
}

func NewTempDir(name string, root string) (TempDir, error) {
	pattern := "flycd"
	if name != "" {
		pattern = name
	}
	tempDir, err := os.MkdirTemp(root, pattern)
	if err != nil {
		return &TempDirImpl{}, fmt.Errorf("error creating temp tempDir: %w", err)
	}
	return &TempDirImpl{WorkDirImpl{
		root: tempDir,
		cwd:  tempDir,
	}}, nil
}

func NewWorkDir(path string) WorkDir {
	return &WorkDirImpl{
		root: path,
		cwd:  path,
	}
}

func (t *WorkDirImpl) NewCommand(command string, args ...string) util_cmd.Command {
	return util_cmd.NewCommandA(command, args...).WithCwd(t.Cwd())
}

func (t *WorkDirImpl) ReadFile(name string) (string, error) {
	data, err := os.ReadFile(t.Cwd() + "/" + name)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", name, err)
	}
	return string(data), nil
}

func (t *WorkDirImpl) WriteFile(name string, contents string) error {
	return os.WriteFile(t.Cwd()+"/"+name, []byte(contents), 0644)
}

func (t *WorkDirImpl) Root() string {
	return t.root
}

func (t *WorkDirImpl) Cwd() string {
	return t.cwd
}

func (t *WorkDirImpl) SetCwd(path string) {
	t.cwd = path
}
