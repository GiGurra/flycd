package flycd

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
)

func isSymLink(dir os.DirEntry) (bool, error) {
	info, err := dir.Info()
	if err != nil {
		return false, fmt.Errorf("error getting symlink info: %w", err)
	}

	return info.Mode().Type() == os.ModeSymlink, nil
}

func canTraverseDir(dir os.DirEntry) (bool, error) {

	symlink, err := isSymLink(dir)
	if err != nil {
		return false, err
	}

	if symlink {
		return false, nil
	}

	if dir.Name() == ".git" || dir.Name() == ".actions" || dir.Name() == ".idea" || dir.Name() == ".vscode" {
		return false, nil
	}

	return true, nil
}

func Deploy(path string) error {
	// Traverse all child dirs and deploy them, as long as they are not symlinks

	println("Deploying:", path)

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	traversableCandidates, hasAppYaml, hasProjectsDir, err := analyseFolder(entries)
	if err != nil {
		return fmt.Errorf("error analysing folder: %w", err)
	}

	if hasAppYaml {

		fmt.Printf("Found app.yaml in %s, deploying app\n", path)
		err2 := deployApp(path)
		if err2 != nil {
			return fmt.Errorf("error deploying app: %w", err2)
		}

	} else if hasProjectsDir {
		println("Found projects dir, traversing only projects")
		return Deploy(path + "/projects")
	} else {
		println("Found neither app.yaml nor projects dir, traversing all dirs")
		for _, entry := range traversableCandidates {
			err := Deploy(path + "/" + entry.Name())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deployApp(path string) error {

	// create a temp tempDir
	tempDir, err := os.MkdirTemp("", "flycd")
	if err != nil {
		return fmt.Errorf("error creating temp tempDir: %w", err)
	}
	defer func(dir string) {
		err := os.RemoveAll(dir)
		if err != nil {
			println("error removing temp tempDir:", err)
		}
	}(tempDir)

	// read the original app.yaml file using a go yaml library
	var cfg AppConfig

	appYaml, err := os.ReadFile(path + "/app.yaml")
	if err != nil {
		return fmt.Errorf("error reading app.yaml from folder %s: %w", path, err)
	}

	err = yaml.Unmarshal(appYaml, &cfg)
	if err != nil {
		return fmt.Errorf("error unmarshalling app.yaml from folder %s: %w", path, err)
	}

	fmt.Printf("app.yaml:\n%+v\n", cfg)

	// Validate the app config
	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("error validating app.yaml from folder %s: %w", path, err)
	}

	// Clone the git repo
	switch cfg.Source.Type {
	case "git":
		_, err = runCommand(tempDir, "git", "clone", cfg.Source.Repo, "repo")
		if err != nil {
			return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
		}
		tempDir = tempDir + "/repo"
	case "local":
		// Copy the local folder to the temp tempDir
		_, err = runCommand(tempDir, "sh", "-c", fmt.Sprintf("cp -r \"%s\" \"%s\"", cfg.Source.Path, tempDir))
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	// Produce the fly.toml file
	_, err = runCommand(tempDir, "sh", "-c", fmt.Sprintf("cat \"%s/fly.toml\"", tempDir))
	if err != nil {
		return fmt.Errorf("error reading fly.toml from folder %s: %w", path, err)
	}

	// execute 'cat app.yaml | yj -yt > fly.toml' on the command line
	cmd := exec.Command("sh", "-c", fmt.Sprintf("cat \"%s/app.yaml\" | yj -yt > \"%s/fly.toml\"", path, tempDir))
	_, err = cmd.Output()

	if err != nil {
		return fmt.Errorf("error producing fly.toml from app.yaml in folder %s: %w", path, err)
	}

	// Now run flyctl and check if the app exists
	_, err = runCommand(tempDir, "flyctl", "status")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "could not find app") {
			println("App not found, creating it")
		} else {
			return fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
		}
	}
	return nil
}

func analyseFolder(entries []os.DirEntry) ([]os.DirEntry, bool, bool, error) {
	// Collect potentially traversable dirs
	traversableCandidates := make([]os.DirEntry, 0)
	hasAppYaml := false
	hasProjectsDir := false

	for _, entry := range entries {
		if entry.IsDir() {
			shouldTraverse, err := canTraverseDir(entry)
			if err != nil {
				return nil, false, false, fmt.Errorf("error checking for symlink: %w", err)
			}
			if !shouldTraverse {
				continue
			}

			if entry.Name() == "projects" {
				hasProjectsDir = true
			}

			traversableCandidates = append(traversableCandidates, entry)

		} else if entry.Name() == "app.yaml" {

			hasAppYaml = true

		}
	}
	return traversableCandidates, hasAppYaml, hasProjectsDir, nil
}

type Command struct {
	App  string
	Args []string
	Cwd  string
}

func runCommand(cwd string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {

		stdErr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdErr = string(exitErr.Stderr)
		}
		return string(out), fmt.Errorf("error running command %s \n %s: %w", command, stdErr, err)
	}

	return string(out), nil
}
