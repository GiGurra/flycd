package flycd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
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

func (r DeployResult) Success() bool {
	return len(r.Failed) == 0
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
			err := DeploySingleAppFromFolder(ctx, app.Path, deployCfg)
			if err != nil {
				result.Failed = append(result.Failed, DeployFailure{
					Spec:  app,
					Cause: err,
				})
				if deployCfg.AbortOnFirstError {
					aborted = true
				}
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

func ExistsApp(ctx context.Context, name string) (bool, error) {
	res, err := util_cmd.
		NewCommand("fly", "status", "-a", name).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(res.Combined), "could not find app") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GetDeployedAppConfig(ctx context.Context, name string) (model.AppConfig, error) {

	// Compare the current deployed appVersion with the new appVersion
	res, err := util_cmd.
		NewCommand("fly", "config", "show", "-a", name).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "no machines configured for this app") {
			return model.AppConfig{Env: map[string]string{}}, nil
		}

		return model.AppConfig{}, fmt.Errorf("error running fly config show for app %s: %w", name, err)
	}

	var deployedCfg model.AppConfig
	err = json.Unmarshal([]byte(res.StdOut), &deployedCfg)
	if err != nil {
		return model.AppConfig{}, fmt.Errorf("error unmarshalling fly config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}
