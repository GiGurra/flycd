package convert

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_toml"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var flags struct {
	force *bool
}

var Cmd = &cobra.Command{
	Use:   "convert <path>",
	Short: "Convert app/apps from fly.toml(s) to/from app.yaml(s)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		argPath := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		path := func() string {
			if filepath.IsAbs(argPath) {
				return argPath
			} else {
				return filepath.Join(cwd, argPath)
			}
		}()

		fmt.Printf("Preparing files inside: %s\n", path)

		// walk the path
		// for each file, check if it's a fly.toml
		// if it is, convert it to app.yaml
		// if it's not, check if it's an app.yaml
		// if it is, convert it to fly.toml
		// if it's not, ignore it

		hasErrs := false

		err = filepath.Walk(path, func(curFilePath string, info os.FileInfo, err error) error {

			// Check if it's a fly.toml
			isFlyToml := info.Name() == "fly.toml" && !info.IsDir()

			if isFlyToml {
				curDirPath := filepath.Dir(curFilePath)
				workDir := util_work_dir.NewWorkDir(curDirPath)
				if workDir.ExistsChild("app.yaml") && !*flags.force {
					fmt.Printf("appl.yaml already esists, skipping conversion @ %s\n", curDirPath)
					return nil
				}

				tomlSrc, err := workDir.ReadFile("fly.toml")
				if err != nil {
					hasErrs = true
					fmt.Printf("Error reading fly.toml @ %s: %v\n", curDirPath, err)
					return nil
				}

				parsed := make(map[string]any)
				err = util_toml.Unmarshal(tomlSrc, &parsed)
				if err != nil {
					hasErrs = true
					fmt.Printf("Error parsing fly.toml @ %s: %v\n", curDirPath, err)
					return nil
				}

				yamlSrc, err := yaml.Marshal(parsed)
				if err != nil {
					hasErrs = true
					fmt.Printf("Error marshalling fly.toml @ %s: %v\n", curDirPath, err)
					return nil
				}

				err = workDir.WriteFile("app.yaml", string(yamlSrc))
				if err != nil {
					hasErrs = true
					fmt.Printf("Error writing app.yaml @ %s: %v\n", curDirPath, err)
					return nil
				}

				fmt.Printf("Converted fly.toml to app.yaml @ %s\n", curDirPath)
			}

			return nil
		})

		if err != nil {
			fmt.Printf("Error walking path %s: %v\n", path, err)
			os.Exit(1)
		}

		if hasErrs {
			fmt.Printf("Errors encountered, see previous logs\n")
			os.Exit(1)
		}

	},
}

func init() {
	flags.force = Cmd.Flags().BoolP("force", "f", false, "Force overwrite of existing files")
}
