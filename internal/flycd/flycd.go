package flycd

import (
	"encoding/json"
	"flycd/internal/flycd/util/util_cmd"
	"flycd/internal/flycd/util/util_tab_table"
	"fmt"
	"github.com/samber/lo"
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

	// While we could do flyctl apps list --json and parse that,
	// This json is huge and slow to fetch. flyctl doesn't seem to offer a
	// simple way to handle this, so I just dump it in table form here instead

	tableStr, err := util_cmd.NewCommand("flyctl", "apps", "list").Run()
	if err != nil {
		return false, err
	}

	parsedTable, err := util_tab_table.ParseTable(tableStr)
	if err != nil {
		return false, fmt.Errorf("error parsing table: %w. tableStr: \n%s", err, tableStr)
	}

	return lo.ContainsBy(parsedTable.RowMaps, func(item map[string]string) bool {
		return item["NAME"] == name
	}), nil
}

func GetDeployedAppConfig(name string) (AppConfig, error) {

	// Compare the current deployed appVersion with the new appVersion
	jsonConf, err := util_cmd.NewCommand("flyctl", "config", "show", "-a", name).Run()
	if err != nil {
		return AppConfig{}, fmt.Errorf("error running flyctl config show for app %s: %w", name, err)
	}

	// TODO: If app is stuck in pending state (prev deploy failed or never performed, we need to repair it...Somehow :S)

	var deployedCfg AppConfig
	err = json.Unmarshal([]byte(jsonConf), &deployedCfg)
	if err != nil {
		return AppConfig{}, fmt.Errorf("error unmarshalling flyctl config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}
