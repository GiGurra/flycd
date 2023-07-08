package flycd

import (
	"encoding/json"
	"flycd/internal/flycd/util/util_cmd"
	"fmt"
	"strings"
)

func Deploy(path string, force bool) error {

	println("Traversing:", path)

	traversableCandidates, hasAppYaml, hasProjectsDir, err := analyseCfgFolder(path)
	if err != nil {
		return fmt.Errorf("error analysing folder: %w", err)
	}

	if hasAppYaml {

		fmt.Printf("Found app.yaml in %s, deploying app\n", path)
		err2 := DeploySingleAppFromFolder(path, force)
		if err2 != nil {
			return fmt.Errorf("error deploying app: %w", err2)
		}

	} else if hasProjectsDir {
		println("Found projects dir, traversing only projects")
		return Deploy(path+"/projects", force)
	} else {
		println("Found neither app.yaml nor projects dir, traversing all dirs")
		for _, entry := range traversableCandidates {
			err := Deploy(path+"/"+entry.Name(), force)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func AppExists(name string) (bool, error) {
	_, err := util_cmd.Run(".", "flyctl", "status", "-a", name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "could not find app") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GetDeployedAppConfig(name string) (AppConfig, error) {

	// Compare the current deployed appVersion with the new appVersion
	jsonConf, err := util_cmd.RunLocal("flyctl", "config", "show", "-a", name)
	if err != nil {
		return AppConfig{}, fmt.Errorf("error running flyctl config show for app %s: %w", name, err)
	}

	var deployedCfg AppConfig
	err = json.Unmarshal([]byte(jsonConf), &deployedCfg)
	if err != nil {
		return AppConfig{}, fmt.Errorf("error unmarshalling flyctl config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}
