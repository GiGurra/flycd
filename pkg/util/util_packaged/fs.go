package util_packaged

import (
	"embed"
	"fmt"
	"github.com/gigurra/flycd/pkg/util/util_work_dir"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type PackagedFile struct {
	Name     string
	Contents string
}

type PackagedFileSystem struct {
	Files       []PackagedFile
	Directories []embed.FS
}

func (embedded PackagedFileSystem) WriteOut(path string) error {
	if path == "" {
		return fmt.Errorf("PackagedFileSystem.WriteOut: path cannot be empty")
	}

	workDir := util_work_dir.NewWorkDir(path)

	fmt.Printf("Writing embedded fs to: %s\n", workDir.Cwd())

	// Write out all files to tempDir
	for _, file := range embedded.Files {

		err := workDir.WriteFile(file.Name, file.Contents)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Name, err)
		}
	}

	// Write out all directories to tempDir
	for _, dir := range embedded.Directories {
		err := writeFS(workDir.Cwd(), dir)
		if err != nil {
			return fmt.Errorf("failed to create directory %v: %w", dir, err)
		}
	}

	return nil
}

func writeFS(outDir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		outPath := filepath.Join(outDir, path)
		if d.IsDir() {
			if err := os.MkdirAll(outPath, 0755); err != nil {
				return err
			}
		} else {
			outFile, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer func(outFile *os.File) {
				err := outFile.Close()
				if err != nil {
					fmt.Printf("Error closing file: %v\n", err)
				}
			}(outFile)

			inFile, err := fsys.Open(path)
			if err != nil {
				return err
			}
			defer func(inFile fs.File) {
				err := inFile.Close()
				if err != nil {
					fmt.Printf("Error closing file: %v\n", err)
				}
			}(inFile)

			_, err = io.Copy(outFile, inFile)
			return err
		}

		return nil
	})
}
