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
		sourcePath := "."
		if cfg.Source.Path != "" {
			sourcePath = cfg.Source.Path
		}
		_, err = runCommand(path, "sh", "-c", fmt.Sprintf("cp -R \"%s/.\" \"%s/\"", sourcePath, tempDir.Cwd))
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	// execute 'cat app.yaml | yj -yt > fly.toml' on the command line
	_, err = tempDir.RunCommand("sh", "-c", "cat app.yaml | yj -yt > fly.toml")
	if err != nil {
		return fmt.Errorf("error producing fly.toml from app.yaml in folder %s: %w", path, err)
	}

	// Now run flyctl and check if the app exists
	_, err = tempDir.RunCommand("flyctl", "status")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "could not find app") {
			println("App not found, creating it")
			err = deployNewApp(tempDir, cfg.LaunchParams)
		} else {
			return fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
		}
	}

	fmt.Printf("not implemented")
	os.Exit(0)

	return err
}

func deployNewApp(tempDir TempDir, lanchParams []string) error {
	allParams := append([]string{"launch"}, lanchParams...)
	_, err := tempDir.RunCommand("flyctl", allParams...)
	return err
}

func deployExistingApp(tempDir TempDir) error {

	return fmt.Errorf("deployExistingApp not implemented")
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
