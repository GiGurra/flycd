package flycd

import (
	"fmt"
)

func Deploy(path string, force bool) error {

	println("Traversing:", path)

	traversableCandidates, hasAppYaml, hasProjectsDir, err := analyseFolder(path)
	if err != nil {
		return fmt.Errorf("error analysing folder: %w", err)
	}

	if hasAppYaml {

		fmt.Printf("Found app.yaml in %s, deploying app\n", path)
		err2 := deployApp(path, force)
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
