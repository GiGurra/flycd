package flycd

import (
	"context"
	"encoding/json"
	"flycd/internal/flycd/util/util_cmd"
	"fmt"
	"strings"
	"time"
)

type DeployFailure struct {
	Spec  SpecNode
	Cause error
}

type DeployResult struct {
	Succeeded []SpecNode
	Failed    []DeployFailure
}

func NewDeployResult() DeployResult {
	return DeployResult{
		Succeeded: make([]SpecNode, 0),
		Failed:    make([]DeployFailure, 0),
	}
}

var SkippedNotValid = fmt.Errorf("skipped: not a valid app")
var SkippedAbortedEarlier = fmt.Errorf("skipped: job aborted earlier")

func DeployAll(
	ctx context.Context,
	path string,
	deployCfg DeployConfig,
) (DeployResult, error) {

	result := NewDeployResult()

	apps, err := ScanForApps(path)
	if err != nil {
		return result, fmt.Errorf("Error finding apps: %w\n", err)
	}

	aborted := false
	for _, app := range apps {
		fmt.Printf("Considering app %s @ %s\n", app.AppConfig.App, app.Path)
		if aborted {
			fmt.Printf("Aborted earlier, skipping!\n")
			result.Failed = append(result.Failed, DeployFailure{
				Spec:  app,
				Cause: SkippedAbortedEarlier,
			})
			continue
		}
		if app.IsValidApp() {
			err := Deploy(ctx, app.Path, deployCfg)
			if err != nil {
				result.Failed = append(result.Failed, DeployFailure{
					Spec:  app,
					Cause: err,
				})
				aborted = true
				fmt.Printf("Error deploying %s @ %s: %v\n:", app.AppConfig.App, app.Path, err)
			}
		} else {
			fmt.Printf("App is NOT valid, skipping!\n")
			result.Failed = append(result.Failed, DeployFailure{
				Spec:  app,
				Cause: SkippedNotValid,
			})
		}
	}

	return result, nil
}

func Deploy(ctx context.Context, path string, deployCfg DeployConfig) error {

	println("Traversing:", path)

	analysis, err := analyseTraversalCandidate(path)
	if err != nil {
		return fmt.Errorf("error analysing folder: %w", err)
	}

	if analysis.HasAppYaml {

		fmt.Printf("Found app.yaml in %s, deploying app\n", path)
		err2 := DeploySingleAppFromFolder(ctx, path, deployCfg)
		if err2 != nil {
			return fmt.Errorf("error deploying app: %w", err2)
		}

	} else if analysis.HasProjectsDir {
		println("Found projects dir, traversing only projects")
		return Deploy(ctx, path+"/projects", deployCfg)
	} else {
		println("Found neither app.yaml nor projects dir, traversing all dirs")
		for _, entry := range analysis.TraversableCandidates {
			err := Deploy(ctx, path+"/"+entry.Name(), deployCfg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func AppExists(ctx context.Context, name string) (bool, error) {
	res, err := util_cmd.
		NewCommand("flyctl", "status", "-a", name).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(10).
		Run(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(res.Combined), "could not find app") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GetDeployedAppConfig(ctx context.Context, name string) (AppConfig, error) {

	// Compare the current deployed appVersion with the new appVersion
	res, err := util_cmd.
		NewCommand("flyctl", "config", "show", "-a", name).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(10).
		Run(ctx)
	if err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "no machines configured for this app") {
			return AppConfig{Env: map[string]string{}}, nil
		}

		return AppConfig{}, fmt.Errorf("error running flyctl config show for app %s: %w", name, err)
	}

	var deployedCfg AppConfig
	err = json.Unmarshal([]byte(res.StdOut), &deployedCfg)
	if err != nil {
		return AppConfig{}, fmt.Errorf("error unmarshalling flyctl config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}
