package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_git"
	"github.com/gigurra/flycd/internal/flycd/util/util_toml"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"github.com/google/uuid"
	"golang.org/x/mod/sumdb/dirhash"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DeployConfig struct {
	Force             bool
	Retries           int
	AttemptTimeout    time.Duration
	AbortOnFirstError bool
}

func NewDeployConfig() DeployConfig {
	return DeployConfig{
		Force:             false,
		Retries:           2,
		AttemptTimeout:    5 * time.Minute,
		AbortOnFirstError: true,
	}
}

func (c DeployConfig) WithAbortOnFirstError(state ...bool) DeployConfig {
	if len(state) > 0 {
		c.AbortOnFirstError = state[0]
	} else {
		c.AbortOnFirstError = true
	}
	return c
}

func (c DeployConfig) WithForce(force ...bool) DeployConfig {
	if len(force) > 0 {
		c.Force = force[0]
	} else {
		c.Force = true
	}
	return c
}

func (c DeployConfig) WithRetries(retries ...int) DeployConfig {
	if len(retries) > 0 {
		c.Retries = retries[0]
	} else {
		c.Retries = 5
	}
	return c
}

func (c DeployConfig) WithAttemptTimeout(timeout ...time.Duration) DeployConfig {
	if len(timeout) > 0 {
		c.AttemptTimeout = timeout[0]
	} else {
		c.AttemptTimeout = 5 * time.Minute
	}
	return c
}

func DeployAppFromInlineConfig(ctx context.Context, deployCfg DeployConfig, cfg model.AppConfig) (SingleAppDeploySuccessType, error) {

	cfgDir, err := util_work_dir.NewTempDir(cfg.App, "")
	if err != nil {
		return "", fmt.Errorf("error creating deployment temp dir: %w", err)
	}
	defer cfgDir.RemoveAll()

	yamlBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("error marshalling app config: %w", err)
	}

	err = cfgDir.WriteFile("app.yaml", string(yamlBytes))
	if err != nil {
		return "", fmt.Errorf("error writing app.yaml: %w", err)
	}

	return DeploySingleAppFromFolder(ctx, cfgDir.Root(), deployCfg)
}

type SingleAppDeploySuccessType string

const (
	SingleAppDeployCreated  SingleAppDeploySuccessType = "created"
	SingleAppDeployUpdated  SingleAppDeploySuccessType = "updated"
	SingleAppDeployNoChange SingleAppDeploySuccessType = "no-change"
)

func DeploySingleAppFromFolder(ctx context.Context, path string, deployCfg DeployConfig) (SingleAppDeploySuccessType, error) {

	cfgDir := util_work_dir.NewWorkDir(path)

	cfg, err := readAppConfig(path)
	if err != nil {
		return "", err
	}

	cfgHash, err := dirhash.HashDir(cfgDir.Cwd(), "", dirhash.DefaultHash)
	if err != nil {
		return "", fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
	}

	tempDir, err := util_work_dir.NewTempDir(cfg.App, "")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %w", err)
	}
	defer tempDir.RemoveAll()

	appHash, err := newUUIDString()
	if err != nil {
		return "", fmt.Errorf("error generating uuid: %w", err)
	}

	switch cfg.Source.Type {
	case model.SourceTypeGit:

		cloneResult, err := util_git.CloneShallow(ctx, cfg.Source, tempDir)
		if err != nil {
			return "", fmt.Errorf("cloning git repo: %w", err)
		}

		tempDir = cloneResult.Dir
		appHash = cloneResult.Hash

	case model.SourceTypeLocal:
		srcDir := func() util_work_dir.WorkDir {
			if filepath.IsAbs(cfg.Source.Path) {
				return cfgDir.WithRootFsCwd(cfg.Source.Path)
			} else {
				return cfgDir.WithChildCwd(cfg.Source.Path)
			}
		}()

		// check if srcDir exists
		if !srcDir.Exists() {
			// Try with it as an absolute path
			fmt.Printf("Local path '%s' does not exist, trying as absolute path\n", cfg.Source.Path)
			srcDir = util_work_dir.NewWorkDir(cfg.Source.Path)
			if !srcDir.Exists() {
				return "", fmt.Errorf("local path '%s' does not exist", cfg.Source.Path)
			}
		}

		err = srcDir.CopyContentsTo(tempDir)
		if err != nil {
			return "", fmt.Errorf("error copying local folder %s: %w", srcDir.Cwd(), err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}

		if err != nil {
			return "", fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	case model.SourceTypeInlineDockerFile:
		// Copy the local folder to the temp tempDir
		err := tempDir.WriteFile("Dockerfile", cfg.Source.Inline)
		if err != nil {
			return "", fmt.Errorf("error writing Dockerfile: %w", err)
		}
		err = tempDir.WriteFile(".dockerignore", "")
		if err != nil {
			return "", fmt.Errorf("error writing .dockerignore: %w", err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}

	default:
		return "", fmt.Errorf("unknown source type %s", cfg.Source.Type)
	}

	appHash = strings.TrimSpace(appHash)
	cfg.Env["FLYCD_CONFIG_VERSION"] = cfgHash
	cfg.Env["FLYCD_APP_VERSION"] = appHash
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF_BRANCH"] = cfg.Source.Ref.Branch
	cfg.Env["FLYCD_APP_SOURCE_REF_COMMIT"] = cfg.Source.Ref.Commit
	cfg.Env["FLYCD_APP_SOURCE_REF_TAG"] = cfg.Source.Ref.Tag

	// Write a new app.yaml file with the appHash
	cfgBytesYaml, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("error marshalling app.yaml: %w", err)
	}

	cfgBytesToml, err := util_toml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("error marshalling fly.toml: %w", err)
	}

	err = tempDir.WriteFile("app.yaml", string(cfgBytesYaml))
	if err != nil {
		return "", fmt.Errorf("error writing app.yaml: %w", err)
	}

	err = tempDir.WriteFile("fly.toml", cfgBytesToml)
	if err != nil {
		return "", fmt.Errorf("error writing fly.toml: %w", err)
	}

	// Create a docker ignore file matching git ignore, if a docker ignore file doesn't already exist
	if !tempDir.ExistsChild(".dockerignore") {
		// Check if a git ignore file exists
		if tempDir.ExistsChild(".gitignore") {
			// Copy the git ignore file to docker ignore
			err = tempDir.CopyFile(".gitignore", ".dockerignore")
			if err != nil {
				return "", fmt.Errorf("error copying .gitignore to .dockerignore: %w", err)
			}
		} else {
			// No git ignore file, so create an empty docker ignore file
			err = tempDir.WriteFile(".dockerignore", "")
			if err != nil {
				return "", fmt.Errorf("error writing .dockerignore: %w", err)
			}
		}
	}

	// Now run fly.io cli and check if the app exists
	fmt.Printf("Checking if the app %s exists\n", cfg.App)
	appExists, err := ExistsApp(ctx, cfg.App)
	if err != nil {
		return "", fmt.Errorf("error running fly status in folder %s: %w", path, err)
	}

	if appExists {
		fmt.Printf("App %s exists, grabbing its currently deployed config from fly.io\n", cfg.App)
		deployedCfg, err := GetDeployedAppConfig(ctx, cfg.App)
		if err != nil {
			return "", fmt.Errorf("error getting deployed app config: %w", err)
		}

		fmt.Printf("Comparing deployed config with current config\n")
		if deployCfg.Force ||
			deployedCfg.Env["FLYCD_APP_VERSION"] != appHash ||
			deployedCfg.Env["FLYCD_CONFIG_VERSION"] != cfgHash {
			fmt.Printf("App %s needs to be re-deployed, doing it now!\n", cfg.App)
			err = deployExistingApp(ctx, cfg, tempDir, deployCfg)
			if err != nil {
				return "", err
			}
			return SingleAppDeployUpdated, nil
		} else {
			println("App is already up to date, skipping deploy")
			return SingleAppDeployNoChange, nil
		}
	} else {
		println("App not found, creating it")
		err = createNewApp(ctx, cfg, tempDir, true)
		if err != nil {
			return "", fmt.Errorf("error creating new app: %w", err)
		}
		println("Issuing an explicit deploy command, since a fly.io bug when deploying within the launch freezes the operation")
		err = deployExistingApp(ctx, cfg, tempDir, deployCfg)
		if err != nil {
			return "", err
		}
		return SingleAppDeployCreated, nil
	}
}

func createNewApp(
	ctx context.Context,
	cfg model.AppConfig,
	tempDir util_work_dir.WorkDir,
	twoStep bool,
) error {
	allParams := append([]string{"launch"}, cfg.LaunchParams...)
	allParams = append(allParams, "--remote-only")
	if twoStep {
		allParams = append(allParams, "--build-only")
	}
	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error creating app %s: %w", cfg.App, err)
	}
	return nil
}

func deployExistingApp(
	ctx context.Context,
	cfg model.AppConfig,
	tempDir util_work_dir.WorkDir,
	deployCfg DeployConfig,
) error {
	allParams := append([]string{"deploy"}, cfg.DeployParams...)
	allParams = append(allParams, "--remote-only", "--detach")

	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithTimeout(deployCfg.AttemptTimeout).
		WithTimeoutRetries(deployCfg.Retries).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error deploying app %s: %w", cfg.App, err)
	}
	return nil
}

func readAppConfig(path string) (model.AppConfig, error) {
	var cfg model.AppConfig

	appYaml, err := os.ReadFile(path + "/app.yaml")
	if err != nil {
		return model.AppConfig{}, fmt.Errorf("error reading app.yaml from folder %s: %w", path, err)
	}

	err = yaml.Unmarshal(appYaml, &cfg)
	if err != nil {
		return model.AppConfig{}, fmt.Errorf("error unmarshalling app.yaml from folder %s: %w", path, err)
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
