package flycd

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"strings"
)

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
