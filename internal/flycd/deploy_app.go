package flycd

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

func deployApp(path string) error {

	tempDir, err := NewTempDir()
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer tempDir.Remove()

	cfg, err := readAppConfig(path, err)
	if err != nil {
		return err
	}

	switch cfg.Source.Type {
	case "git":
		_, err = tempDir.RunCommand("git", "clone", cfg.Source.Repo, "repo")
		if err != nil {
			return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
		}
		tempDir.Cwd = tempDir.Cwd + "/repo"
	case "local":
		// Copy the local folder to the temp tempDir
		_, err = tempDir.RunCommand("sh", "-c", fmt.Sprintf("cp -r \"%s\" \"%s\"", cfg.Source.Path, tempDir))
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	// Produce the fly.toml file
	_, err = tempDir.RunCommand("sh", "-c", fmt.Sprintf("cat \"%s/fly.toml\"", tempDir))
	if err != nil {
		return fmt.Errorf("error reading fly.toml from folder %s: %w", path, err)
	}

	// execute 'cat app.yaml | yj -yt > fly.toml' on the command line
	_, err = tempDir.RunCommand("sh", "-c", fmt.Sprintf("cat \"%s/app.yaml\" | yj -yt > \"%s/fly.toml\"", path, tempDir))
	if err != nil {
		return fmt.Errorf("error producing fly.toml from app.yaml in folder %s: %w", path, err)
	}

	// Now run flyctl and check if the app exists
	_, err = tempDir.RunCommand("flyctl", "status")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "could not find app") {
			println("App not found, creating it")
		} else {
			return fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
		}
	}
	return nil
}

func readAppConfig(path string, err error) (AppConfig, error) {
	var cfg AppConfig

	appYaml, err := os.ReadFile(path + "/app.yaml")
	if err != nil {
		return AppConfig{}, fmt.Errorf("error reading app.yaml from folder %s: %w", path, err)
	}

	err = yaml.Unmarshal(appYaml, &cfg)
	if err != nil {
		return AppConfig{}, fmt.Errorf("error unmarshalling app.yaml from folder %s: %w", path, err)
	}

	err = cfg.Validate()
	if err != nil {
		return cfg, fmt.Errorf("error validating app.yaml from folder %s: %w", path, err)
	}

	return cfg, nil
}
