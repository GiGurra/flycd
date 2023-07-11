package flycd

import (
	"context"
	"fmt"
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

func DeployAppFromInlineConfig(ctx context.Context, deployCfg DeployConfig, cfg AppConfig) error {

	cfgDir, err := util_work_dir.NewTempDir(cfg.App, "")
	if err != nil {
		return fmt.Errorf("error creating deployment temp dir: %w", err)
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

	return DeploySingleAppFromFolder(ctx, cfgDir.Root(), deployCfg)
}

func DeploySingleAppFromFolder(ctx context.Context, path string, deployCfg DeployConfig) error {

	cfgDir := util_work_dir.NewWorkDir(path)

	cfg, err := readAppConfig(path)
	if err != nil {
		return err
	}

	cfgHash, err := dirhash.HashDir(cfgDir.Cwd(), "", dirhash.DefaultHash)
	if err != nil {
		return fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
	}

	tempDir, err := util_work_dir.NewTempDir(cfg.App, "")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer tempDir.RemoveAll()

	appHash, err := newUUIDString()
	if err != nil {
		return fmt.Errorf("error generating uuid: %w", err)
	}

	switch cfg.Source.Type {
	case SourceTypeGit:

		var err error

		if cfg.Source.Ref.Commit != "" {
			// Shallow clone of specific commit
			// https://stackoverflow.com/questions/31278902/how-to-shallow-clone-a-specific-commit-with-depth-1
			_, err = tempDir.
				NewCommand("git", "init").
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error initializing git repo: %w", err)
			}

			_, err = tempDir.
				NewCommand("git", "remote", "add", "origin", cfg.Source.Repo).
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error adding git remote: %w", err)
			}

			_, err = tempDir.
				NewCommand("git", "fetch", "--depth", "1", "origin", cfg.Source.Ref.Commit).
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error fetching git commit: %w", err)
			}

			_, err = tempDir.
				NewCommand("git", "checkout", "FETCH_HEAD").
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error checking out git commit: %w", err)
			}

		} else if cfg.Source.Ref.Tag != "" {
			_, err = tempDir.
				NewCommand("git", "clone", cfg.Source.Repo, "repo", "--depth", "1", "--branch", cfg.Source.Ref.Tag).
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
			}
			tempDir = tempDir.WithPushCwd("repo")

		} else if cfg.Source.Ref.Branch != "" {
			_, err = tempDir.
				NewCommand("git", "clone", cfg.Source.Repo, "repo", "--depth", "1", "--branch", cfg.Source.Ref.Branch).
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
			}
			tempDir = tempDir.WithPushCwd("repo")
		} else {
			_, err = tempDir.NewCommand("git", "clone", cfg.Source.Repo, "repo", "--depth", "1").
				WithStdLogging().
				Run(ctx)
			if err != nil {
				return fmt.Errorf("error cloning git repo %s: %w", cfg.Source.Repo, err)
			}
			tempDir = tempDir.WithPushCwd("repo")
		}

		res, err := tempDir.
			NewCommand("git", "rev-parse", "HEAD").
			Run(ctx)
		if err != nil {
			return fmt.Errorf("error getting git commit hash: %w", err)
		}
		appHash = strings.TrimSpace(res.StdOut)
	case SourceTypeLocal:
		srcDir := cfgDir.WithPushCwd(cfg.Source.Path)

		// check if srcDir exists
		if !srcDir.Exists() {
			// Try with it as an absolute path
			fmt.Printf("Local path '%s' does not exist, trying as absolute path\n", cfg.Source.Path)
			srcDir = util_work_dir.NewWorkDir(cfg.Source.Path)
			if !srcDir.Exists() {
				return fmt.Errorf("local path '%s' does not exist", cfg.Source.Path)
			}
		}

		err = srcDir.CopyContentsTo(tempDir)
		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", srcDir.Cwd(), err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}

		if err != nil {
			return fmt.Errorf("error copying local folder %s: %w", cfg.Source.Path, err)
		}
	case SourceTypeInlineDockerFile:
		// Copy the local folder to the temp tempDir
		err := tempDir.WriteFile("Dockerfile", cfg.Source.Inline)
		if err != nil {
			return fmt.Errorf("error writing Dockerfile: %w", err)
		}
		err = tempDir.WriteFile(".dockerignore", "")
		if err != nil {
			return fmt.Errorf("error writing .dockerignore: %w", err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}
		if err != nil {
			return fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
		}

	default:
		return fmt.Errorf("unknown source type %s", cfg.Source.Type)
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
		return fmt.Errorf("error marshalling app.yaml: %w", err)
	}

	cfgBytesToml, err := util_toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshalling fly.toml: %w", err)
	}

	err = tempDir.WriteFile("app.yaml", string(cfgBytesYaml))
	if err != nil {
		return fmt.Errorf("error writing app.yaml: %w", err)
	}

	err = tempDir.WriteFile("fly.toml", cfgBytesToml)
	if err != nil {
		return fmt.Errorf("error writing fly.toml: %w", err)
	}

	// Create a docker ignore file matching git ignore, if a docker ignore file doesn't already exist
	wantedDockerIgnoreFilePath := filepath.Join(tempDir.Cwd(), ".dockerignore")
	if _, err := os.Stat(wantedDockerIgnoreFilePath); os.IsNotExist(err) {
		// Check if a git ignore file exists
		gitIgnoreFilePath := filepath.Join(tempDir.Cwd(), ".gitignore")
		if _, err := os.Stat(gitIgnoreFilePath); os.IsNotExist(err) {
			// No git ignore file, so create an empty docker ignore file
			err = tempDir.WriteFile(".dockerignore", "")
			if err != nil {
				return fmt.Errorf("error writing .dockerignore: %w", err)
			}
		} else {
			// Copy the git ignore file to docker ignore
			err = tempDir.CopyFile(".gitignore", ".dockerignore")
			if err != nil {
				return fmt.Errorf("error copying .gitignore to .dockerignore: %w", err)
			}
		}
	}

	// Now run flyctl and check if the app exists
	fmt.Printf("Checking if the app %s exists\n", cfg.App)
	appExists, err := AppIsDeployed(ctx, cfg.App)
	if err != nil {
		return fmt.Errorf("error running flyctl status in folder %s: %w", path, err)
	}

	if appExists {
		fmt.Printf("App %s exists, grabbing its currently deployed config from fly.io\n", cfg.App)
		deployedCfg, err := GetDeployedAppConfig(ctx, cfg.App)
		if err != nil {
			return fmt.Errorf("error getting deployed app config: %w", err)
		}

		fmt.Printf("Comparing deployed config with current config\n")
		if deployCfg.Force ||
			deployedCfg.Env["FLYCD_APP_VERSION"] != appHash ||
			deployedCfg.Env["FLYCD_CONFIG_VERSION"] != cfgHash {
			fmt.Printf("App %s needs to be re-deployed, doing it now!\n", cfg.App)
			return deployExistingApp(ctx, cfg, tempDir, deployCfg)
		} else {
			println("App is already up to date, skipping deploy")
		}
	} else {
		println("App not found, creating it")
		err = createNewApp(ctx, cfg, tempDir, true)
		if err != nil {
			return fmt.Errorf("error creating new app: %w", err)
		}
		println("Issuing an explicit deploy command, since a fly.io bug when deploying within the launch freezes the operation")
		return deployExistingApp(ctx, cfg, tempDir, deployCfg)
	}

	return nil
}

func createNewApp(
	ctx context.Context,
	cfg AppConfig,
	tempDir util_work_dir.WorkDir,
	twoStep bool,
) error {
	allParams := append([]string{"launch"}, cfg.LaunchParams...)
	allParams = append(allParams, "--remote-only")
	if twoStep {
		allParams = append(allParams, "--build-only")
	}
	_, err := tempDir.
		NewCommand("flyctl", allParams...).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(10).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error creating app %s: %w", cfg.App, err)
	}
	return nil
}

func deployExistingApp(
	ctx context.Context,
	cfg AppConfig,
	tempDir util_work_dir.WorkDir,
	deployCfg DeployConfig,
) error {
	allParams := append([]string{"deploy"}, cfg.DeployParams...)
	allParams = append(allParams, "--remote-only", "--detach")

	_, err := tempDir.
		NewCommand("flyctl", allParams...).
		WithTimeout(deployCfg.AttemptTimeout).
		WithTimeoutRetries(deployCfg.Retries).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error deploying app %s: %w", cfg.App, err)
	}
	return nil
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
