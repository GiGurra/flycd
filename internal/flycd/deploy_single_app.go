package flycd

import (
	"flycd/internal/flycd/util/util_work_dir"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

func DeployAppFromConfig(cfg AppConfig, force bool) error {

	cfgDir, err := util_work_dir.NewTempDir(cfg.App, ".")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer cfgDir.RemoveAll()

	yamlBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error marshalling app config: %w", err)
	}

	err = cfgDir.WriteFile("app.yaml", string(yamlBytes))
	if err != nil {
		return fmt.Errorf("error writing app.yaml: %w", err)
	}

	return DeploySingleAppFromFolder(cfgDir.Root, force)
}

func DeploySingleAppFromFolder(path string, force bool) error {

	cfgDir := util_work_dir.NewWorkDir(path)

	cfg, err := readAppConfig(path)
	if err != nil {
		return err
	}

	tempDir, err := util_work_dir.NewTempDir(cfg.App, "")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer tempDir.RemoveAll()

	cfgVersion, err := cfgDir.RunCommand("git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("error getting git commit hash of cfg dir: %w", err)
	}

	appVersion, err := newUUIDString()
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

		appVersion, err = tempDir.RunCommand("git", "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}
	case SourceTypeLocal:
		// Copy the local folder to the temp tempDir
		sourcePath := "."
		if cfg.Source.Path != "" {
			sourcePath = cfg.Source.Path
		}

		appVersion, err = cfgDir.RunCommand("sh", "-c", fmt.Sprintf("git rev-parse HEAD"))
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}

		_, err = cfgDir.RunCommand("sh", "-c", fmt.Sprintf("cp -R \"%s/.\" \"%s/\"", sourcePath, tempDir.Cwd))
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	case SourceTypeInlineDockerFile:
		// Copy the local folder to the temp tempDir
		err := tempDir.WriteFile("Dockerfile", cfg.Source.InlineDockerFile)
		if err != nil {
			return fmt.Errorf("error writing Dockerfile: %w", err)
		}

		appVersion, err = cfgDir.RunCommand("sh", "-c", fmt.Sprintf("git rev-parse HEAD"))
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}

	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	appVersion = strings.TrimSpace(appVersion)
	cfg.Env["FLYCD_CONFIG_VERSION"] = cfgVersion
	cfg.Env["FLYCD_APP_VERSION"] = appVersion
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF"] = cfg.Source.Ref

	// Write a new app.yaml file with the appVersion
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshalling app.yaml: %w", err)
	}

	err = tempDir.WriteFile("app.yaml", string(cfgBytes))

	// execute 'cat app.yaml | yj -yt > fly.toml' on the util_cmd line
	_, err = tempDir.RunCommand("sh", "-c", "cat app.yaml | yj -yt > fly.toml")
	if err != nil {
		return fmt.Errorf("error producing fly.toml from app.yaml in folder %s: %w", path, err)
	}

	// Now run flyctl and check if the app exists
	appExists, err := AppExists(cfg.App)
	if err != nil {
		return fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
	}

	if appExists {
		deployedCfg, err := GetDeployedAppConfig(cfg.App)
		if err != nil {
			return fmt.Errorf("error getting deployed app config: %w", err)
		}

		if force ||
			deployedCfg.Env["FLYCD_APP_VERSION"] != appVersion ||
			deployedCfg.Env["FLYCD_CONFIG_VERSION"] != cfgVersion {
			return deployExistingApp(tempDir, cfg.DeployParams)
		} else {
			println("App is already up to date, skipping deploy")
		}
	} else {
		println("App not found, creating it")
		return deployNewApp(tempDir, cfg.LaunchParams)
	}

	fmt.Printf("not implemented")
	os.Exit(0)

	return err
}

func deployNewApp(tempDir util_work_dir.WorkDir, launchParams []string) error {
	allParams := append([]string{"launch"}, launchParams...)
	err := tempDir.RunCommandStreamedPassthrough("flyctl", allParams...)
	return err
}

func deployExistingApp(tempDir util_work_dir.WorkDir, deployParams []string) error {
	allParams := append([]string{"deploy"}, deployParams...)
	_, err := tempDir.RunCommand("flyctl", allParams...)
	return err
}

func readAppConfig(path string) (AppConfig, error) {
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
	result, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
