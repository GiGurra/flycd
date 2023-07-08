package flycd

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
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

	// random uuid
	version, err := newUUIDString()
	if err != nil {
		return fmt.Errorf("error generating uuid: %w", err)
	}

	switch cfg.Source.Type {
	case SourceTypeGit:
		_, err = tempDir.RunCommand("git", "clone", cfg.Source.Repo, "repo")
		if err != nil {
			return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
		}
		tempDir.Cwd = tempDir.Cwd + "/repo"

		version, err = tempDir.RunCommand("git", "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}
	case SourceTypeLocal:
		// Copy the local folder to the temp tempDir
		sourcePath := "."
		if cfg.Source.Path != "" {
			sourcePath = cfg.Source.Path
		}

		version, err = runCommand(path, "sh", "-c", fmt.Sprintf("git rev-parse HEAD"))
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}

		_, err = runCommand(path, "sh", "-c", fmt.Sprintf("cp -R \"%s/.\" \"%s/\"", sourcePath, tempDir.Cwd))
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	version = strings.TrimSpace(version)
	cfg.Env["FLYCD_APP_VERSION"] = version
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF"] = cfg.Source.Ref

	// Write a new app.yaml file with the version
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshalling app.yaml: %w", err)
	}

	err = tempDir.WriteFile("app.yaml", string(cfgBytes))

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
			err = fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
		}
	} else {
		println("App found, updating it")

		// Compare the current deployed version with the new version
		jsonConf, err := tempDir.RunCommand("flyctl", "config", "show")
		if err != nil {
			return fmt.Errorf("error running flyctl config show in folder %s: %w", path, err)
		}

		var deployedCfg AppConfig
		err = json.Unmarshal([]byte(jsonConf), &deployedCfg)
		if err != nil {
			return fmt.Errorf("error unmarshalling flyctl config show in folder %s: %w", path, err)
		}

		if deployedCfg.Env["FLYCD_APP_VERSION"] == version {
			println("App is already up to date, skipping deploy")
		} else {
			err = deployExistingApp(tempDir, cfg.DeployParams)
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

func deployExistingApp(tempDir TempDir, deployParams []string) error {
	allParams := append([]string{"deploy"}, deployParams...)
	_, err := tempDir.RunCommand("flyctl", allParams...)
	return err
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

func newUUIDString() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}
