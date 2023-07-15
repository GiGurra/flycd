package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/fly_client"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_git"
	"github.com/gigurra/flycd/internal/flycd/util/util_math"
	"github.com/gigurra/flycd/internal/flycd/util/util_toml"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"github.com/samber/lo"
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

type DeployServiceImpl struct {
	flyClient fly_client.FlyClient
}

func (d DeployServiceImpl) DeployAll(ctx context.Context, path string, deployCfg model.DeployConfig) (model.DeployResult, error) {
	return deployAll(d.flyClient, ctx, path, deployCfg)
}

func (d DeployServiceImpl) DeployAppFromInlineConfig(ctx context.Context, deployCfg model.DeployConfig, cfg model.AppConfig) (model.SingleAppDeploySuccessType, error) {
	return deployAppFromInlineConfig(d.flyClient, ctx, deployCfg, cfg)
}

func (d DeployServiceImpl) DeployAppFromFolder(ctx context.Context, path string, deployCfg model.DeployConfig) (model.SingleAppDeploySuccessType, error) {
	return deployAppFromFolder(d.flyClient, ctx, path, deployCfg)
}

// prove that DeployServiceImpl implements DeployService
var _ DeployService = DeployServiceImpl{}

func NewDeployService(flyClient fly_client.FlyClient) DeployService {
	return DeployServiceImpl{
		flyClient: flyClient,
	}
}

func deployAll(
	flyClient fly_client.FlyClient,
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
				res, err := deployAppFromFolder(flyClient, ctx, appNode.Path, deployCfg)
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

func deployAppFromInlineConfig(
	flyClient fly_client.FlyClient,
	ctx context.Context,
	deployCfg model.DeployConfig,
	cfg model.AppConfig,
) (model.SingleAppDeploySuccessType, error) {

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

	return deployAppFromFolder(flyClient, ctx, cfgDir.Root(), deployCfg)
}

func deployAppFromFolder(
	flyClient fly_client.FlyClient,
	ctx context.Context,
	path string,
	deployCfg model.DeployConfig,
) (model.SingleAppDeploySuccessType, error) {

	cfgDir := util_work_dir.NewWorkDir(path)

	cfgTyped, cfgUntyped, err := readAppConfigs(path)
	if err != nil {
		return "", err
	}

	cfgHash, err := dirhash.HashDir(cfgDir.Cwd(), "", dirhash.DefaultHash)
	if err != nil {
		return "", fmt.Errorf("error getting local dir hash for '%s': %w", path, err)
	}

	tempDir, err := util_work_dir.NewTempDir(cfgTyped.App, "")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %w", err)
	}
	defer tempDir.RemoveAll()

	appHash, err := fetchAppFs(ctx, cfgTyped, cfgDir, &tempDir)
	if err != nil {
		return "", fmt.Errorf("error preparing fs to deploy: %w", err)
	}

	err = mergeCfgAndAppFs(cfgTyped, cfgDir, tempDir)
	if err != nil {
		return "", fmt.Errorf("error merging config and app fs: %w", err)
	}

	updateCfgHashes(&cfgTyped, &cfgUntyped, appHash, cfgHash)

	err = writeOutUpdatedConfigFiles(cfgUntyped, tempDir)
	if err != nil {
		return "", fmt.Errorf("error writing out updated config files: %w", err)
	}

	err = ensureDockerIgnoreExists(tempDir, err)
	if err != nil {
		return "", fmt.Errorf("error ensuring docker ignore exists: %w", err)
	}

	input := deployInput{
		ctx:       ctx,
		flyClient: flyClient,
		deployCfg: deployCfg,
		cfgTyped:  cfgTyped,
		tempDir:   tempDir,
		appHash:   appHash,
		cfgHash:   cfgHash,
	}

	return deployAppToFly(input)
}

type deployInput struct {
	ctx       context.Context
	flyClient fly_client.FlyClient
	deployCfg model.DeployConfig
	cfgTyped  model.AppConfig
	tempDir   util_work_dir.WorkDir
	appHash   string
	cfgHash   string
}

func runIntermediateSteps(input deployInput) error {

	err := runIntermediateVolumeSteps(input)
	if err != nil {
		return fmt.Errorf("error running intermediate volume steps: %w", err)
	}

	// add intermediate steps here

	return nil
}

// runIntermediateVolumeSteps Here we analyse the deployed state
// of volumes for this app vs the desired state and bring the
// deployed state up to the desired state
func runIntermediateVolumeSteps(input deployInput) error {

	if len(input.cfgTyped.Volumes) == 0 {
		return nil // no volumes for this app
	}

	allDeployedVolumes, err := input.flyClient.GetAppVolumes(input.ctx, input.cfgTyped.App)
	if err != nil {
		return fmt.Errorf("error getting deployed volumes for app %s: %w", input.cfgTyped.App, err)
	}

	deployedVolumesByNameAndRegion := lo.GroupBy(allDeployedVolumes, func(volume model.VolumeState) string {
		return volume.Name + volume.Region
	})

	minVolumeCountByServicesPerRegion, err := getMinimumVolumeCountPerRegion(input)
	if err != nil {
		return fmt.Errorf("error getting minimum volume count for app %s: %w", input.cfgTyped.App, err)
	}

	numExtendedVolumes := 0
	numCreatedVolumes := 0

	for _, region := range input.cfgTyped.RegionsWPrimaryLast() {

		for _, wantedVolume := range input.cfgTyped.Volumes {

			wantedCount := util_math.Max(wantedVolume.Count, minVolumeCountByServicesPerRegion[region])

			fmt.Printf("Volumes '%s': We need %d x %d GB in region %s \n", wantedVolume.Name, wantedCount, wantedVolume.SizeGb, region)

			deployedVolumesThisRegion := deployedVolumesByNameAndRegion[wantedVolume.Name+region]

			// First bring all deployed volumes up to our required size
			for _, currentVolume := range deployedVolumesThisRegion {
				if currentVolume.SizeGb < wantedVolume.SizeGb {
					fmt.Printf("Resizing app %s's volume %s from %d to %d in region %s\n", input.cfgTyped.App, currentVolume.Name, currentVolume.SizeGb, wantedVolume.SizeGb, region)
					err := input.flyClient.ExtendVolume(input.ctx, input.cfgTyped.App, currentVolume.ID, wantedVolume.SizeGb)
					if err != nil {
						return fmt.Errorf("error resizing volume %s for app %s in region %s: %w", currentVolume.ID, input.cfgTyped.App, region, err)
					}
					numExtendedVolumes++
				}
			}

			// Create new needed volumes
			newVolumesNeeded := util_math.Max(0, wantedCount-len(deployedVolumesThisRegion))
			for i := 0; i < newVolumesNeeded; i++ {
				fmt.Printf("Creating new %s volume for app %s in region %s \n", wantedVolume.Name, input.cfgTyped.App, region)
				_, err := input.flyClient.CreateVolume(input.ctx, input.cfgTyped.App, wantedVolume, region)
				if err != nil {
					return fmt.Errorf("error creating volume %s for app %s in region %s: %w", wantedVolume.Name, input.cfgTyped.App, region, err)
				}
				numCreatedVolumes++
			}
		}
	}

	if numExtendedVolumes > 0 || numCreatedVolumes > 0 {
		fmt.Printf("Extended %d volumes and created %d new volumes for app %s \n", numExtendedVolumes, numCreatedVolumes, input.cfgTyped.App)
	} else {
		fmt.Printf("No change of volumes needed for app %s \n", input.cfgTyped.App)
	}

	return nil
}

func getMinimumVolumeCountPerRegion(input deployInput) (map[string]int, error) {
	// We need at least as many volumes as the minimum number of app instances.
	// In the fly.io configuration, this is given by the `min_instances` field.
	minSvcReq := input.cfgTyped.MinMachinesRunning()

	result := map[string]int{}

	// We should also consider the actual number of app instances that are currently running.
	scales, err := input.flyClient.GetAppScale(input.ctx, input.cfgTyped.App)
	if err != nil {
		return result, fmt.Errorf("error getting app scales for app %s: %w", input.cfgTyped.App, err)
	}

	for _, region := range input.cfgTyped.RegionsWPrimaryLast() {

		// count app processes
		appProcessScales := lo.Map(scales, func(scale model.ScaleState, _ int) int {
			if scale.Process == "app" && scale.IncludesRegion(region) {
				return scale.CountInRegion(region)
			} else {
				return 0
			}
		})
		appProcessCount := lo.Reduce(appProcessScales, func(agg int, item int, _ int) int {
			return agg + item
		}, 0)
		if appProcessCount > minSvcReq {
			minSvcReq = appProcessCount
		}

		result[region] = minSvcReq
	}

	return result, nil
}

func deployAppToFly(
	input deployInput,
) (model.SingleAppDeploySuccessType, error) {

	fmt.Printf("Checking if the app %s exists\n", input.cfgTyped.App)
	appExists, err := input.flyClient.ExistsApp(input.ctx, input.cfgTyped.App)
	if err != nil {
		return "", fmt.Errorf("error checking if app %s exists: %w", input.cfgTyped.App, err)
	}

	if appExists {
		fmt.Printf("App %s exists, grabbing its currently deployed config from fly.io\n", input.cfgTyped.App)
		deployedCfg, err := input.flyClient.GetDeployedAppConfig(input.ctx, input.cfgTyped.App)
		if err != nil {
			return "", fmt.Errorf("error getting deployed app config: %w", err)
		}

		fmt.Printf("Comparing deployed config with current config\n")
		if input.deployCfg.Force ||
			deployedCfg.Env["FLYCD_APP_VERSION"] != input.appHash ||
			deployedCfg.Env["FLYCD_CONFIG_VERSION"] != input.cfgHash {
			fmt.Printf("App %s needs to be re-deployed, doing it now!\n", input.cfgTyped.App)
			err = runIntermediateSteps(input) // set up volumes etc
			if err != nil {
				return "", err
			}
			for _, region := range input.cfgTyped.RegionsWPrimaryLast() {
				fmt.Printf("Deploying app %s to region %s\n", input.cfgTyped.App, region)
				err = input.flyClient.DeployExistingApp(input.ctx, input.cfgTyped, input.tempDir, input.deployCfg, region)
				if err != nil {
					return "", err
				}
			}
			return model.SingleAppDeployUpdated, nil
		} else {
			fmt.Printf("App is already up to date, skipping deploy\n")
			return model.SingleAppDeployNoChange, nil
		}
	} else {
		fmt.Printf("App not found, creating it\n")
		err = input.flyClient.CreateNewApp(input.ctx, input.cfgTyped, input.tempDir, true)
		if err != nil {
			return "", fmt.Errorf("error creating new app: %w", err)
		}
		err = runIntermediateSteps(input) // set up volumes etc
		if err != nil {
			return "", err
		}
		fmt.Printf("Issuing an explicit deploy command, since a fly.io bug when deploying within the launch freezes the operation\n")
		for _, region := range input.cfgTyped.RegionsWPrimaryLast() {
			fmt.Printf("Deploying app %s to region %s\n", input.cfgTyped.App, region)
			err = input.flyClient.DeployExistingApp(input.ctx, input.cfgTyped, input.tempDir, input.deployCfg, region)
		}
		if err != nil {
			return "", err
		}
		return model.SingleAppDeployCreated, nil
	}
}

func fetchAppFs(
	ctx context.Context,
	cfgTyped model.AppConfig,
	cfgDir util_work_dir.WorkDir,
	tempDir *util_work_dir.WorkDir,
) ( /* appHash */ string, error) {
	appHash := ""

	switch cfgTyped.Source.Type {
	case model.SourceTypeGit:

		cloneResult, err := util_git.CloneShallow(ctx, cfgTyped.Source, *tempDir)
		if err != nil {
			return "", fmt.Errorf("cloning git repo: %w", err)
		}

		*tempDir = tempDir.WithRootFsCwd(cloneResult.Dir.Cwd())
		appHash = cloneResult.Hash

	case model.SourceTypeLocal:
		srcDir := func() util_work_dir.WorkDir {
			if filepath.IsAbs(cfgTyped.Source.Path) {
				return cfgDir.WithRootFsCwd(cfgTyped.Source.Path)
			} else {
				return cfgDir.WithChildCwd(cfgTyped.Source.Path)
			}
		}()

		// check if srcDir exists
		if !srcDir.Exists() {
			// Try with it as an absolute path
			fmt.Printf("Local path '%s' does not exist, trying as absolute path\n", cfgTyped.Source.Path)
			srcDir = util_work_dir.NewWorkDir(cfgTyped.Source.Path)
			if !srcDir.Exists() {
				return "", fmt.Errorf("local path '%s' does not exist", cfgTyped.Source.Path)
			}
		}

		err := srcDir.CopyContentsTo(*tempDir)
		if err != nil {
			return "", fmt.Errorf("error copying local folder %s: %w", srcDir.Cwd(), err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", tempDir.Cwd(), err)
		}

		if err != nil {
			return "", fmt.Errorf("error copying local folder %s: %w", cfgTyped.Source.Path, err)
		}
	case model.SourceTypeInlineDockerFile:
		// Copy the local folder to the temp tempDir
		err := tempDir.WriteFile("Dockerfile", cfgTyped.Source.Inline)
		if err != nil {
			return "", fmt.Errorf("error writing Dockerfile: %w", err)
		}
		err = tempDir.WriteFile(".dockerignore", "")
		if err != nil {
			return "", fmt.Errorf("error writing .dockerignore: %w", err)
		}

		appHash, err = dirhash.HashDir(tempDir.Cwd(), "", dirhash.DefaultHash)
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", tempDir.Cwd(), err)
		}
		if err != nil {
			return "", fmt.Errorf("error getting local dir hash for '%s': %w", tempDir.Cwd(), err)
		}

	default:
		return "", fmt.Errorf("unknown source type %s", cfgTyped.Source.Type)
	}
	appHash = strings.TrimSpace(appHash) // Not sure if we need this anymore

	return appHash, nil
}

func mergeCfgAndAppFs(
	cfg model.AppConfig,
	cfgDir util_work_dir.WorkDir,
	tempDir util_work_dir.WorkDir,
) error {
	// Check if to copy config contents to tempDir
	if cfg.MergeCfg.All {
		err := cfgDir.CopyContentsTo(tempDir)
		if err != nil {
			return fmt.Errorf("could not copy config dir contents to cloned repo dir for %+v: %w", cfg, err)
		}
	} else if len(cfg.MergeCfg.Include) > 0 {
		for _, exactPath := range cfg.MergeCfg.Include {
			err := cfgDir.CopyFile(exactPath, tempDir.Cwd()+"/"+exactPath)
			if err != nil {
				return fmt.Errorf("could not copy config file '%s' to cloned repo dir for %+v: %w", exactPath, cfg, err)
			}
		}
	}
	return nil
}

func updateCfgHashes(
	cfg *model.AppConfig,
	cfgUntyped *map[string]any,
	appHash string,
	cfgHash string,
) {
	cfg.Env["FLYCD_CONFIG_VERSION"] = cfgHash
	cfg.Env["FLYCD_APP_VERSION"] = appHash
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF_BRANCH"] = cfg.Source.Ref.Branch
	cfg.Env["FLYCD_APP_SOURCE_REF_COMMIT"] = cfg.Source.Ref.Commit
	cfg.Env["FLYCD_APP_SOURCE_REF_TAG"] = cfg.Source.Ref.Tag
	(*cfgUntyped)["env"] = cfg.Env
}

func writeOutUpdatedConfigFiles(cfgUntyped map[string]any, tempDir util_work_dir.WorkDir) error {
	cfgBytesYaml, err := yaml.Marshal(cfgUntyped)
	if err != nil {
		return fmt.Errorf("error marshalling app.yaml: %w", err)
	}

	cfgBytesToml, err := util_toml.Marshal(cfgUntyped)
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
	return nil
}

func ensureDockerIgnoreExists(tempDir util_work_dir.WorkDir, err error) error {
	// Create a docker ignore file matching git ignore, if a docker ignore file doesn't already exist
	// If we don't do this, fly.io cli will get stuck waiting or user input
	if !tempDir.ExistsChild(".dockerignore") {
		// Check if a git ignore file exists
		if tempDir.ExistsChild(".gitignore") {
			// Copy the git ignore file to docker ignore
			err = tempDir.CopyFile(".gitignore", ".dockerignore")
			if err != nil {
				return fmt.Errorf("error copying .gitignore to .dockerignore: %w", err)
			}
		} else {
			// No git ignore file, so create an empty docker ignore file
			err = tempDir.WriteFile(".dockerignore", "")
			if err != nil {
				return fmt.Errorf("error writing .dockerignore: %w", err)
			}
		}
	}
	return nil
}

func readAppConfigs(path string) (model.AppConfig, map[string]any, error) {
	typed, err := readAppConfigTyped(path)
	if err != nil {
		return model.AppConfig{}, nil, err
	}
	untyped, err := readAppConfigUntyped(path)
	if err != nil {
		return model.AppConfig{}, nil, err
	}

	return typed, untyped, nil
}

func readAppConfigTyped(path string) (model.AppConfig, error) {
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
