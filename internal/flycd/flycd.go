package flycd

import (
	"context"
	"encoding/json"
	"flycd/internal/flycd/util/util_cmd"
	"fmt"
	"strings"
	"time"
)

func Deploy(ctx context.Context, path string, deployCfg DeployConfig) error {

	println("Traversing:", path)

	analysis, err := analyseSingleFolder(path)
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
