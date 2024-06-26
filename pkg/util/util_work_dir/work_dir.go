package util_work_dir

import (
	"fmt"
	"github.com/GiGurra/cmder"
	cp "github.com/otiai10/copy"
	"os"
	"path/filepath"
)

type WorkDir struct {
	root string
	cwd  string
}

func (t WorkDir) RemoveAll() {
	err := os.RemoveAll(t.Root())
	if err != nil {
		fmt.Printf("error removing dir %s: %s", t.Root(), err)
	}
}

func NewTempDir(name string, root string) (WorkDir, error) {
	tempDir, err := os.MkdirTemp(root, name)
	if err != nil {
		return WorkDir{}, fmt.Errorf("error creating temp tempDir: %w", err)
	}
	return WorkDir{
		root: tempDir,
		cwd:  tempDir,
	}, nil
}

func NewWorkDir(path string) WorkDir {
	return WorkDir{
		root: path,
		cwd:  path,
	}
}

func (t WorkDir) NewCommand(command string, args ...string) cmder.Spec {
	return cmder.NewA(command, args...).WithWorkingDirectory(t.Cwd())
}

func (t WorkDir) ReadFile(name string) (string, error) {
	data, err := os.ReadFile(t.Cwd() + "/" + name)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %w", name, err)
	}
	return string(data), nil
}

func (t WorkDir) WriteFile(name string, contents string) error {
	return os.WriteFile(t.Cwd()+"/"+name, []byte(contents), 0644)
}

func (t WorkDir) Root() string {
	return t.root
}

func (t WorkDir) Cwd() string {
	return t.cwd
}

func (t WorkDir) WithChildCwd(path string) WorkDir {

	if path == "" {
		return t
	}

	if path == "." {
		return t
	}

	t.cwd = filepath.Join(t.cwd, path)
	return t
}

func (t WorkDir) WithRootFsCwd(path string) WorkDir {
	t.cwd = path
	return t
}

func (t WorkDir) CopyContentsTo(target WorkDir) error {
	err := cp.Copy(t.Cwd(), target.Cwd())
	if err != nil {
		return fmt.Errorf("error copying contents from %s to %s: %w", t.Cwd(), target.Cwd(), err)
	}
	return nil
}

func (t WorkDir) Exists() bool {
	_, err := os.Stat(t.Cwd())
	return err == nil
}

func (t WorkDir) ExistsChild(path string) bool {
	_, err := os.Stat(filepath.Join(t.Cwd(), path))
	return err == nil
}

func (t WorkDir) CopyFile(from string, to string) error {
	fromAbs := filepath.Join(t.Cwd(), from)
	toAbs := filepath.Join(t.Cwd(), to)
	return cp.Copy(fromAbs, toAbs)
}
