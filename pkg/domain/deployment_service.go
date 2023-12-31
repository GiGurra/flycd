package domain

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/ext/fly_client"
	"github.com/gigurra/flycd/pkg/util/util_cvt"
	"github.com/gigurra/flycd/pkg/util/util_git"
	"github.com/gigurra/flycd/pkg/util/util_math"
	"github.com/gigurra/flycd/pkg/util/util_toml"
	"github.com/gigurra/flycd/pkg/util/util_work_dir"
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
		preCalculatedAppConfig *model.PreCalculatedAppConfig,
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

func (d DeployServiceImpl) DeployAppFromFolder(
	ctx context.Context,
	path string,
	deployCfg model.DeployConfig,
	preCalculatedAppConfig *model.PreCalculatedAppConfig,
) (model.SingleAppDeploySuccessType, error) {
	return deployAppFromFolder(d.flyClient, ctx, path, deployCfg, preCalculatedAppConfig)
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

	err := TraverseDeepAppTree(path, model.TraverseAppTreeContext{
		Context: ctx,
		ValidAppCb: func(ctx model.TraverseAppTreeContext, appNode model.AppAtFsNode) error {
			fmt.Printf("Considering app %s @ %s\n", appNode.AppConfig.App, appNode.Path)
			if deployCfg.AbortOnFirstError && result.HasErrors() {
				fmt.Printf("Aborted earlier, skipping!\n")
				result.FailedApps = append(result.FailedApps, model.AppDeployFailure{
					Spec:  appNode,
					Cause: SkippedAbortedEarlier,
				})
				return nil
			} else {
				res, err := deployAppFromFolder(flyClient, ctx, appNode.Path, deployCfg, appNode.ToPreCalculatedApoConf())
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
		InvalidAppCb: func(ctx model.TraverseAppTreeContext, appNode model.AppAtFsNode) error {
			result.FailedApps = append(result.FailedApps, model.AppDeployFailure{
				Spec:  appNode,
				Cause: SkippedNotValid(appNode.ErrCause()),
			})
			return nil
		},
		BeginProjectCb: func(ctx model.TraverseAppTreeContext, projNode model.ProjectAtFsNode) error {
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

	untypedCfg := map[string]any{}
	err = yaml.Unmarshal(yamlBytes, &untypedCfg)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling app config: %w", err)
	}

	err = cfgDir.WriteFile("app.yaml", string(yamlBytes))
	if err != nil {
		return "", fmt.Errorf("error writing app.yaml: %w", err)
	}

	return deployAppFromFolder(flyClient, ctx, cfgDir.Cwd(), deployCfg, &model.PreCalculatedAppConfig{
		Typed:   cfg,
		UnTyped: untypedCfg,
	})
}

func deployAppFromFolder(
	flyClient fly_client.FlyClient,
	ctx context.Context,
	path string,
	deployCfg model.DeployConfig,
	preCalculatedAppCfg *model.PreCalculatedAppConfig,
) (model.SingleAppDeploySuccessType, error) {

	if preCalculatedAppCfg != nil {
		err := preCalculatedAppCfg.Typed.Validate()
		if err != nil {
			return "", fmt.Errorf("error validating app config: %w", err)
		}
	}

	cfgDir := util_work_dir.NewWorkDir(path)

	cfgTyped, cfgUntyped, err := func() (model.AppConfig, map[string]any, error) {
		if preCalculatedAppCfg != nil {
			return preCalculatedAppCfg.Typed, preCalculatedAppCfg.UnTyped, nil
		} else {
			return readAppConfigs(path)
		}
	}()
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

	err = runIntermediateSecretsSteps(input)
	if err != nil {
		return fmt.Errorf("error running intermediate secrets steps: %w", err)
	}

	err = runIntermediateNetworkingSteps(input)
	if err != nil {
		return fmt.Errorf("error running intermediate networking steps: %w", err)
	}

	// add intermediate steps here

	return nil
}

func runPostDeploySteps(input deployInput) error {

	deployedScales, err := input.flyClient.GetAppScale(input.ctx, input.cfgTyped.App)
	if err != nil {
		return fmt.Errorf("error getting app scale for app %s: %w", input.cfgTyped.App, err)
	}

	err = runScaleCountAllRegionsPostDeployStep(input, deployedScales)
	if err != nil {
		return fmt.Errorf("error during runScaleCountAllRegionsPostDeployStep step: %w", err)
	}

	err = runScaleRamPostDeployStep(input, deployedScales)
	if err != nil {
		return fmt.Errorf("error during runScaleRamPostDeployStep step: %w", err)
	}

	err = runScaleVmPostDeployStep(input, deployedScales)
	if err != nil {
		return fmt.Errorf("error during runScaleVmPostDeployStep step: %w", err)
	}

	// add post-deploy steps here

	return nil
}

func runScaleCountAllRegionsPostDeployStep(input deployInput, deployedScales []model.ScaleState) error {

	fmt.Printf("Checking if we need to scale up instance count in any region\n")
	minSvcReq := input.cfgTyped.MinMachinesFromServices()
	if len(input.cfgTyped.ExtraRegions) == 0 &&
		minSvcReq <= 1 &&
		len(input.cfgTyped.Machines.CountPerRegion) == 0 &&
		input.cfgTyped.Machines.Count <= 1 {
		fmt.Printf("No need to scale up instance count in any region, beacuse we only have one region and don't require more than 1 instance\n")
		return nil // nothing to do
	}

	var err error

	currentCountPerRegion := model.CountDeployedAppsPerRegion(deployedScales)
	wantedRegions := input.cfgTyped.RegionsWPrimaryLast()
	for _, wantedRegion := range wantedRegions {
		wantedCountForRegion := input.cfgTyped.Machines.CountInRegion(wantedRegion)
		if wantedCountForRegion < minSvcReq {
			wantedCountForRegion = minSvcReq
		}

		if wantedCountForRegion > currentCountPerRegion[wantedRegion] {
			fmt.Printf("Need to region %s has %d instances, but we want %d (region_min)... scaling up!\n", wantedRegion, currentCountPerRegion[wantedRegion], wantedCountForRegion)
			err = input.flyClient.ScaleApp(input.ctx, input.cfgTyped.App, wantedRegion, wantedCountForRegion)
			if err != nil {
				// Don't return immediately, try to scale all regions
				fmt.Printf("error scaling app %s to %d in region %s: %v\n", input.cfgTyped.App, wantedCountForRegion, wantedRegion, err)
			}
		} else {
			fmt.Printf("region %s has %d instances, which is >= %d (region_min)... no need to scale up\n", wantedRegion, currentCountPerRegion[wantedRegion], wantedCountForRegion)
		}
	}

	return err
}

func runScaleVmPostDeployStep(input deployInput, deployedScales []model.ScaleState) error {

	fmt.Printf("Checking if we need to change vm type\n")
	if input.cfgTyped.Machines.CpuCores <= 0 {
		fmt.Printf("No need to change vm type, no vm type specified\n")
		return nil
	}

	cpuType := input.cfgTyped.Machines.CpuType
	if cpuType == "" {
		fmt.Printf("Cpu type unspecified, defaulting to whatever is already deployed\n")
	}

	currentScalesByName := lo.GroupBy(deployedScales, func(scale model.ScaleState) string {
		return scale.Process
	})

	needToScale := false
	for _, scale := range currentScalesByName["app"] {
		if (cpuType != "" && scale.CPUKind != cpuType) || scale.CPUs != input.cfgTyped.Machines.CpuCores {
			needToScale = true
			if cpuType == "" {
				cpuType = scale.CPUKind
			}
			break
		}
	}

	if needToScale {
		fmt.Printf("Scaling app %s to %s with %d cores\n", input.cfgTyped.App, cpuType, input.cfgTyped.Machines.CpuCores)
		vmString := fmt.Sprintf("%s-cpu-%dx", cpuType, input.cfgTyped.Machines.CpuCores)
		err := input.flyClient.ScaleAppVm(input.ctx, input.cfgTyped.App, vmString)
		if err != nil {
			return fmt.Errorf("error scaling app %s to %s with %d cores: %w", input.cfgTyped.App, cpuType, input.cfgTyped.Machines.CpuCores, err)
		} else {
			fmt.Printf("scaled app %s to %s with %d cores\n", input.cfgTyped.App, cpuType, input.cfgTyped.Machines.CpuCores)
		}
	} else {
		fmt.Printf("No need to scale app %s to %s with %d cores, either already at that level, or 'app' process not found\n", input.cfgTyped.App, cpuType, input.cfgTyped.Machines.CpuCores)
	}

	return nil
}

func runScaleRamPostDeployStep(input deployInput, deployedScales []model.ScaleState) error {

	fmt.Printf("Checking if we need to change amount of ram per instance\n")
	if input.cfgTyped.Machines.RamMB <= 0 {
		fmt.Printf("No need to change ram per instance, no ram specified\n")
		return nil
	}

	currentScalesByName := lo.GroupBy(deployedScales, func(scale model.ScaleState) string {
		return scale.Process
	})

	needToScale := false
	for _, scale := range currentScalesByName["app"] {
		if scale.MemoryMB != input.cfgTyped.Machines.RamMB {
			needToScale = true
			break
		}
	}

	if needToScale {
		fmt.Printf("Scaling app %s to %d ram\n", input.cfgTyped.App, input.cfgTyped.Machines.RamMB)
		err := input.flyClient.ScaleAppRam(input.ctx, input.cfgTyped.App, input.cfgTyped.Machines.RamMB)
		if err != nil {
			return fmt.Errorf("error scaling app %s to %d ram: %w", input.cfgTyped.App, input.cfgTyped.Machines.RamMB, err)
		} else {
			fmt.Printf("scaled app %s to %d ram\n", input.cfgTyped.App, input.cfgTyped.Machines.RamMB)
		}
	} else {
		fmt.Printf("No need to scale app %s to %d ram, either already at that level, or 'app' process not found\n", input.cfgTyped.App, input.cfgTyped.Machines.RamMB)
	}

	return nil
}

// runIntermediateSecretsSteps Here we extract and deploy all secrets
func runIntermediateSecretsSteps(input deployInput) error {

	if len(input.cfgTyped.Secrets) == 0 {
		return nil
	}

	// we re-save all secrets every time, since we don't know which ones have changed
	secretsToSave := []fly_client.Secret{}
	for _, secretRef := range input.cfgTyped.Secrets {
		secretValue, err := secretRef.GetSecretValue()
		if err != nil {
			return fmt.Errorf("error getting value for secret %s for app %s: %w", secretRef.Name, input.cfgTyped.App, err)
		}
		secretsToSave = append(secretsToSave, fly_client.Secret{
			Name:  secretRef.Name,
			Value: secretValue,
		})
	}
	err := input.flyClient.SaveSecrets(input.ctx, input.cfgTyped.App, secretsToSave, true)
	if err != nil {
		return fmt.Errorf("error saving secrets for app %s: %w", input.cfgTyped.App, err)
	}
	return nil
}

// runIntermediateSecretsSteps Here we extract and deploy all secrets
func runIntermediateNetworkingSteps(input deployInput) error {

	networkCfg := input.cfgTyped.NetworkConfig

	if networkCfg.IsEmpty() {
		return nil
	}

	currentIps, err := input.flyClient.ListIps(input.ctx, input.cfgTyped.App)
	if err != nil {
		return fmt.Errorf("error getting ips for app %s: %w", input.cfgTyped.App, err)
	}

	toBeKept := map[string]bool{}

	// First, add all missing IPs
	for _, cfgIp := range networkCfg.Ips {
		err := cfgIp.Validate()
		if err != nil {
			return fmt.Errorf("error validating ip config %+v: %w", cfgIp, err)
		}
		needCreate := true
		for _, existIp := range currentIps {

			if existIp.Ipv() != cfgIp.V {
				continue
			}

			if existIp.IsPrivate() != cfgIp.Private {
				continue
			}

			if strings.ToLower(existIp.Network) != strings.ToLower(cfgIp.Network) {
				continue
			}

			if strings.ToLower(existIp.Region) != strings.ToLower(cfgIp.Region) {
				if cfgIp.Region == "" && strings.ToLower(existIp.Region) == "global" {
					// this is fine
				} else {
					continue
				}
			}

			// Found an IP we should keep. No need to create it.
			toBeKept[existIp.Id] = true
			needCreate = false
			break
		}

		if needCreate {
			fmt.Printf("Creating ip %+v for app %s\n", cfgIp, input.cfgTyped.App)
			err := input.flyClient.CreateIp(input.ctx, input.cfgTyped.App, cfgIp)
			if err != nil {
				return fmt.Errorf("error creating ip %+v for app %s: %w", cfgIp, input.cfgTyped.App, err)
			}
		}
	}

	// Release/prune unspecified IPs
	if networkCfg.AutoPruneIps {
		fmt.Printf("Pruning ips for app %s\n", input.cfgTyped.App)
		for _, ip := range currentIps {
			if !toBeKept[ip.Id] {
				fmt.Printf("Removing ip %s for app %s\n", ip.Id, input.cfgTyped.App)
				err = input.flyClient.DeleteIp(input.ctx, input.cfgTyped.App, ip.Id, ip.Address)
				if err != nil {
					return fmt.Errorf("error pruning ip %s for app %s: %w", ip.Address, input.cfgTyped.App, err)
				}
			}
		}
	} else {
		fmt.Printf("Not pruning ips for app %s\n", input.cfgTyped.App)
	}

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

			fmt.Printf("Currently deployed volumes: %d\n", len(deployedVolumesThisRegion))
			for _, deployedVolume := range deployedVolumesThisRegion {
				fmt.Printf(" - %s (%d GB)\n", deployedVolume.Name, deployedVolume.SizeGb)
			}

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
	minReqBase := util_math.Max(1, input.cfgTyped.MinMachinesFromServices())

	// We also need at least as many volumes as the minimum number of app instances
	machineCfg := input.cfgTyped.Machines

	// We should also consider the actual number of app instances that are currently running.
	scales, err := input.flyClient.GetAppScale(input.ctx, input.cfgTyped.App)
	if err != nil {
		return map[string]int{}, fmt.Errorf("error getting app scales for app %s: %w", input.cfgTyped.App, err)
	}

	result := model.CountDeployedAppsPerRegion(scales)
	for region, count := range result {
		if count < minReqBase {
			result[region] = minReqBase
		}
		if wantedMachineCountInRegion := machineCfg.CountInRegion(region); count < wantedMachineCountInRegion {
			result[region] = wantedMachineCountInRegion
		}
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
			// At first, it was believed that we could deploy to each region here to create machines there.
			// However, it turns out that fly.io doesn't create machines at all with the 'deploy' command at all.
			// So there is no point in looping over regions here and deploying to each.
			fmt.Printf("Deploying app %s\n", input.cfgTyped.App)
			err = input.flyClient.DeployExistingApp(input.ctx, input.cfgTyped, input.tempDir, input.deployCfg, input.cfgTyped.PrimaryRegion)
			if err != nil {
				return "", err
			}
			err = runPostDeploySteps(input)
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
		err = input.flyClient.CreateNewApp(input.ctx, input.cfgTyped, input.tempDir, true)
		if err != nil {
			return "", fmt.Errorf("error creating new app: %w", err)
		}
		err = runIntermediateSteps(input) // set up volumes etc
		if err != nil {
			return "", err
		}
		fmt.Printf("Issuing an explicit deploy command\n")
		// At first, it was believed that we could deploy to each region here to create machines there.
		// However, it turns out that fly.io doesn't create machines at all with the 'deploy' command at all.
		// So there is no point in looping over regions here and deploying to each.
		fmt.Printf("Deploying app %s\n", input.cfgTyped.App)
		err = input.flyClient.DeployExistingApp(input.ctx, input.cfgTyped, input.tempDir, input.deployCfg, input.cfgTyped.PrimaryRegion)
		if err != nil {
			return "", err
		}
		err = runPostDeploySteps(input)
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

		cloneResult, err := util_git.CloneShallow(ctx, cfgTyped.Source.AsGitCloneSource(), *tempDir)
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
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	cfg.Env["FLYCD_CONFIG_VERSION"] = cfgHash
	cfg.Env["FLYCD_APP_VERSION"] = appHash
	cfg.Env["FLYCD_APP_SOURCE_TYPE"] = string(cfg.Source.Type)
	cfg.Env["FLYCD_APP_SOURCE_PATH"] = cfg.Source.Path
	cfg.Env["FLYCD_APP_SOURCE_REPO"] = cfg.Source.Repo
	cfg.Env["FLYCD_APP_SOURCE_REF_BRANCH"] = cfg.Source.Ref.Branch
	cfg.Env["FLYCD_APP_SOURCE_REF_COMMIT"] = cfg.Source.Ref.Commit
	cfg.Env["FLYCD_APP_SOURCE_REF_TAG"] = cfg.Source.Ref.Tag
	envUntyped, err := util_cvt.StructToMapYaml(cfg.Env)
	if err != nil {
		panic(err)
	}
	(*cfgUntyped)["env"] = envUntyped
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

func readAppConfigs(
	path string,
) (model.AppConfig, map[string]any, error) {

	appYaml, err := os.ReadFile(path + "/app.yaml")
	if err != nil {
		return model.AppConfig{}, map[string]any{}, fmt.Errorf("error reading app.yaml from folder %s: %w", path, err)
	}

	typed, untyped, err := model.CommonAppConfig{}.MakeAppConfig(appYaml)
	if err != nil {
		return model.AppConfig{}, untyped, fmt.Errorf("error making app config from folder %s: %w", path, err)
	}

	return typed, untyped, nil
}
