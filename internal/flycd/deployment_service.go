package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/fly_client"
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
)

func SkippedNotValid(cause error) error { return fmt.Errorf("skipped: not a valid app: %w", cause) }

var SkippedAbortedEarlier = fmt.Errorf("skipped: job aborted earlier")

// DeployService We have multiple consumers of this interface from day 1, so it's prob ok to declare it here
type DeployService interface {
	DeployAll(
		ctx context.Context,
		path string,
		deployCfg model.DeployConfig,
	) (model.DeployResult, error)

	DeployAppFromInlineConfig(
		ctx context.Context,
		deployCfg model.DeployConfig,
		cfg model.AppConfig,
	) (model.SingleAppDeploySuccessType, error)

	DeployAppFromFolder(
		ctx context.Context,
		path string,
		deployCfg model.DeployConfig,
	) (model.SingleAppDeploySuccessType, error)
}

type DeployServiceImpl struct{}

func (d DeployServiceImpl) DeployAll(ctx context.Context, path string, deployCfg model.DeployConfig) (model.DeployResult, error) {
	return deployAll(ctx, path, deployCfg)
}

func (d DeployServiceImpl) DeployAppFromInlineConfig(ctx context.Context, deployCfg model.DeployConfig, cfg model.AppConfig) (model.SingleAppDeploySuccessType, error) {
	return deployAppFromInlineConfig(ctx, deployCfg, cfg)
}

func (d DeployServiceImpl) DeployAppFromFolder(ctx context.Context, path string, deployCfg model.DeployConfig) (model.SingleAppDeploySuccessType, error) {
	return deployAppFromFolder(ctx, path, deployCfg)
}

// prove that DeployServiceImpl implements DeployService
var _ DeployService = DeployServiceImpl{}

func NewDeployService() DeployService {
	return DeployServiceImpl{}
}

func deployAll(
	ctx context.Context,
	path string,
	deployCfg model.DeployConfig,
) (model.DeployResult, error) {

	result := model.NewEmptyDeployResult()

	err := TraverseDeepAppTree(ctx, path, model.TraverseAppTreeOptions{
		ValidAppCb: func(appNode model.AppNode) error {
			fmt.Printf("Considering app %s @ %s\n", appNode.AppConfig.App, appNode.Path)
			if deployCfg.AbortOnFirstError && result.HasErrors() {
				fmt.Printf("Aborted earlier, skipping!\n")
				result.FailedApps = append(result.FailedApps, model.AppDeployFailure{
					Spec:  appNode,
					Cause: SkippedAbortedEarlier,
				})
				return nil
			} else {
				res, err := deployAppFromFolder(ctx, appNode.Path, deployCfg)
				if err != nil {
					result.FailedApps = append(result.FailedApps, model.AppDeployFailure{
						Spec:  appNode,
						Cause: err,
					})
				} else {
					result.SucceededApps = append(result.SucceededApps, model.AppDeploySuccess{
						Spec:        appNode,
						SuccessType: res,
					})
				}
				return nil
			}
		},
		InvalidAppCb: func(appNode model.AppNode) error {
			result.FailedApps = append(result.FailedApps, model.AppDeployFailure{
				Spec:  appNode,
				Cause: SkippedNotValid(appNode.ErrCause()),
			})
			return nil
		},
		BeginProjectCb: func(projNode model.ProjectNode) error {
			if deployCfg.AbortOnFirstError && result.HasErrors() {
				result.FailedProjects = append(result.FailedProjects, model.ProjectProcessingFailure{
					Spec:  projNode,
					Cause: SkippedAbortedEarlier,
				})
				return nil
			} else if !projNode.IsValidProject() {
				result.FailedProjects = append(result.FailedProjects, model.ProjectProcessingFailure{
					Spec:  projNode,
					Cause: SkippedNotValid(projNode.ErrCause()),
				})
				return nil
			} else {
				result.ProcessedProjects = append(result.ProcessedProjects, projNode)
				return nil
			}
		},
	})
	if err != nil {
		return result, fmt.Errorf("error traversing app tree: %w", err)
	}
	return result, nil
}

func deployAppFromInlineConfig(ctx context.Context, deployCfg model.DeployConfig, cfg model.AppConfig) (model.SingleAppDeploySuccessType, error) {

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

	return deployAppFromFolder(ctx, cfgDir.Root(), deployCfg)
}

func deployAppFromFolder(ctx context.Context, path string, deployCfg model.DeployConfig) (model.SingleAppDeploySuccessType, error) {

	cfgDir := util_work_dir.NewWorkDir(path)

	// we need to read the conf both typed (comparing state values) and untyped (to not lose any data)

	cfg, err := readAppConfig(path)
	if err != nil {
		return "", err
	}

	cfgUntyped, err := readAppConfigUntyped(path)
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

	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}

	//

	appHash = strings.TrimSpace(appHash)
	cfg.Env["FLYCD_CONFIG_VERSION"] = cfgHash
	cfg.Env["FLYCD_APP_VERSION"] = appHash
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF_BRANCH"] = cfg.Source.Ref.Branch
	cfg.Env["FLYCD_APP_SOURCE_REF_COMMIT"] = cfg.Source.Ref.Commit
	cfg.Env["FLYCD_APP_SOURCE_REF_TAG"] = cfg.Source.Ref.Tag

	cfgUntyped["env"] = cfg.Env

	// Write a new app.yaml file with the appHash
	cfgBytesYaml, err := yaml.Marshal(cfgUntyped)
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
	appExists, err := fly_client.ExistsApp(ctx, cfg.App)
	if err != nil {
		return "", fmt.Errorf("error running fly status in folder %s: %w", path, err)
	}

	if appExists {
		fmt.Printf("App %s exists, grabbing its currently deployed config from fly.io\n", cfg.App)
		deployedCfg, err := fly_client.GetDeployedAppConfig(ctx, cfg.App)
		if err != nil {
			return "", fmt.Errorf("error getting deployed app config: %w", err)
		}

		fmt.Printf("Comparing deployed config with current config\n")
		if deployCfg.Force ||
			deployedCfg.Env["FLYCD_APP_VERSION"] != appHash ||
			deployedCfg.Env["FLYCD_CONFIG_VERSION"] != cfgHash {
			fmt.Printf("App %s needs to be re-deployed, doing it now!\n", cfg.App)
			err = fly_client.DeployExistingApp(ctx, cfg, tempDir, deployCfg)
			if err != nil {
				return "", err
			}
			return model.SingleAppDeployUpdated, nil
		} else {
			fmt.Printf("App is already up to date, skipping deploy\n")
			return model.SingleAppDeployNoChange, nil
		}
	} else {
		fmt.Printf("App not found, creating it\n")
		err = fly_client.CreateNewApp(ctx, cfg, tempDir, true)
		if err != nil {
			return "", fmt.Errorf("error creating new app: %w", err)
		}
		fmt.Printf("Issuing an explicit deploy command, since a fly.io bug when deploying within the launch freezes the operation\n")
		err = fly_client.DeployExistingApp(ctx, cfg, tempDir, deployCfg)
		if err != nil {
			return "", err
		}
		return model.SingleAppDeployCreated, nil
	}
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

func readAppConfigUntyped(path string) (map[string]any, error) {
	cfg := make(map[string]any)

	appYaml, err := os.ReadFile(path + "/app.yaml")
	if err != nil {
		return cfg, fmt.Errorf("error reading app.yaml from folder %s: %w", path, err)
	}

	err = yaml.Unmarshal(appYaml, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("error unmarshalling app.yaml from folder %s: %w", path, err)
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
