package convert

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/util/util_cobra"
	"github.com/gigurra/flycd/pkg/util/util_toml"
	"github.com/gigurra/flycd/pkg/util/util_work_dir"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type flags struct {
	force *bool
}

func (f *flags) Init(cmd *cobra.Command) {
	f.force = cmd.Flags().BoolP("force", "f", false, "Force overwrite of existing files")
}

func Cmd(_ context.Context) *cobra.Command {
	flags := flags{}
	return util_cobra.CreateCmd(&flags, func() *cobra.Command {
		return &cobra.Command{
			Use:   "convert <path>",
			Short: "Convert app/apps from fly.toml(s) to app.yaml(s)",
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

						config := make(map[string]any)
						err = util_toml.Unmarshal(tomlSrc, &config)
						if err != nil {
							hasErrs = true
							fmt.Printf("Error parsing fly.toml @ %s: %v\n", curDirPath, err)
							return nil
						}

						if _, ok := config["source"]; !ok {
							// No source, we need to add a default one
							config["source"] = model.NewLocalFolderSource("")
						}

						if _, ok := config["mounts"]; ok {
							// fly.toml only has a single mount as a map/object in the mounts field :D
							mount, isMap := config["mounts"].(map[string]any)
							if isMap {
								config["mounts"] = []any{mount}
							}
						}

						yamlSrc, err := yaml.Marshal(config)
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
	})
}
