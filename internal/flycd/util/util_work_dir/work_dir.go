package util_work_dir

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
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
	pattern := "flycd"
	if name != "" {
		pattern = name
	}
	tempDir, err := os.MkdirTemp(root, pattern)
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

func (t WorkDir) NewCommand(command string, args ...string) util_cmd.Command {
	return util_cmd.NewCommandA(command, args...).WithCwd(t.Cwd())
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

func (t WorkDir) WithPushCwd(path string) WorkDir {

	if path == "" {
		return t
	}

	if path == "." {
		return t
	}

	t.cwd = filepath.Join(t.cwd, path)
	return t
}

func (t WorkDir) CopyContentsTo(target WorkDir) error {
	err := cp.Copy(t.Cwd(), target.Cwd())
	if err != nil {
		return fmt.Errorf("error copying contents from %s to %s: %w", t.Cwd(), target.Cwd(), err)
	}
	return nil
}
