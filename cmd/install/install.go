package install

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/fly_cli"
	"github.com/gigurra/flycd/internal/flycd"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_packaged"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type Flags struct {
	appName     *string
	orgSlug     *string
	region      *string
	projectPath *string
}

func (f *Flags) Init(cmd *cobra.Command) {
	f.appName = cmd.Flags().StringP("app-name", "a", "", "App name to give flycd in your fly.io account")
	f.orgSlug = cmd.Flags().StringP("org", "o", "", "Slug of the fly.io org to install to")
	f.region = cmd.Flags().StringP("region", "r", "", "Region of the fly.io app to install")
}

func Cmd(packaged util_packaged.PackagedFileSystem) *cobra.Command {

	flags := Flags{}

	cmd := &cobra.Command{
		Use:   "install <flycd app name> <fly.io org slug> <fly.io region>",
		Short: "Install flycd into your fly.io account, listening to webhooks from this cfg repo and your app repos",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, _ []string) {

			var err error

			appName := *flags.appName
			if appName == "" {
				// Ask the user for app name
				fmt.Printf("Enter a name for your flycd app: ")
				_, err = fmt.Scanln(&appName)
				if err != nil {
					fmt.Printf("Error reading app name: %v\n", err)
					os.Exit(1)
				}
			}

			orgSlug := *flags.orgSlug
			if orgSlug == "" {
				// Ask the user for org slug
				fmt.Printf("Enter the slug of the fly.io org to install to: ")
				_, err = fmt.Scanln(&orgSlug)
				if err != nil {
					fmt.Printf("Error reading org slug: %v\n", err)
					os.Exit(1)
				}
			}

			region := *flags.region
			if region == "" {
				// Ask the user for region
				fmt.Printf("Enter the region of the fly.io app to install: ")
				_, err = fmt.Scanln(&region)
				if err != nil {
					fmt.Printf("Error reading region: %v\n", err)
					os.Exit(1)
				}
			}

			cwd, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current working directory: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Installing flycd with app-name='%s', org='%s' \n", appName, orgSlug)

			ctx := context.Background()

			fmt.Printf("Check if app named '%s' already exists\n", appName)
			appExists, err := flycd.ExistsApp(ctx, appName)
			if err != nil {
				fmt.Printf("Error checking if app exists: %v\n", err)
				os.Exit(1)
			}

			if !appExists {

				deployCfg := flycd.NewDeployConfig().WithRetries(0)

				fmt.Printf("Creating a dummy app '%s' to reserve the name\n", appName)
				_, err := flycd.DeployAppFromInlineConfig(ctx, deployCfg, model.AppConfig{
					App:           appName,
					Org:           orgSlug,
					PrimaryRegion: region,
					Source:        model.NewInlineDockerFileSource("FROM nginx:latest"),
					LaunchParams:  model.NewDefaultLaunchParams(appName, orgSlug),
					DeployParams:  model.NewDefaultDeployParams(),
					Services:      []model.Service{model.NewDefaultServiceConfig()},
				})
				if err != nil {
					fmt.Printf("Error creating dummy app: %v\n", err)
					os.Exit(1)
				}
			}

			existsAccessTokenSecret, err := fly_cli.ExistsSecret(ctx, fly_cli.ExistsSecretCmd{
				AppName:    appName,
				SecretName: "FLY_ACCESS_TOKEN",
			})
			if err != nil {
				fmt.Printf("Error checking if access token secret exists: %v\n", err)
				os.Exit(1)
			}

			if !existsAccessTokenSecret {

				fmt.Printf("App name successfully reserved... creating access token for org '%s'\n", orgSlug)
				token, err := fly_cli.CreateOrgToken(ctx, orgSlug)
				if err != nil {
					fmt.Printf("Error creating org token: %v\n", err)
					os.Exit(1)
				}
				token = strings.TrimSpace(token)

				fmt.Printf("Token created.. storing it...\n")

				err = fly_cli.StoreSecret(ctx, fly_cli.StoreSecretCmd{
					AppName:     appName,
					SecretName:  "FLY_ACCESS_TOKEN",
					SecretValue: token,
				})

				if err != nil {
					fmt.Printf("Error storing token: %v\n", err)
					os.Exit(1)
				}

			}

			// Copy flycd sources etc from embedded files to temp dir
			// So we can add it to our docker image, and then build and deploy it
			tempDir, err := util_work_dir.NewTempDir("flycd-install", "")
			if err != nil {
				fmt.Printf("Error creating temp dir: %v\n", err)
				os.Exit(1)
			}
			defer tempDir.RemoveAll()
			err = packaged.WriteOut(tempDir.Cwd())
			if err != nil {
				fmt.Printf("Error writing embedded files: %v\n", err)
				os.Exit(1)
			}

			// Copy cwd/projects to tempDir
			projectsDir := fmt.Sprintf("%s/projects", cwd)
			if _, err := os.Stat(projectsDir); err == nil {
				fmt.Printf("Copying projects dir to temp dir %s...\n", tempDir.Cwd())
				err = cp.Copy(projectsDir, fmt.Sprintf("%s/projects", tempDir.Cwd()))
				if err != nil {
					fmt.Printf("Error copying projects dir: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("No projects dir found in cwd...Not sure what to install.. assuming you will add one later :S\n")
				// Create an empty projects dir in tempDir
				err = os.MkdirAll(fmt.Sprintf("%s/projects", tempDir.Cwd()), 0755)
				if err != nil {
					fmt.Printf("Error creating empty projects dir: %v\n", err)
					os.Exit(1)
				}
			}

			// Deploy it!
			fmt.Printf("Deploying flycd in monitoring mode to fly.io\n")
			deployCfg := flycd.NewDeployConfig().WithForce(true).WithRetries(0)
			_, err = flycd.DeployAppFromInlineConfig(ctx, deployCfg, model.AppConfig{
				App:           appName,
				Org:           orgSlug,
				PrimaryRegion: region,
				Source:        model.NewLocalFolderSource(tempDir.Cwd()),
				LaunchParams:  model.NewDefaultLaunchParams(appName, orgSlug),
				DeployParams:  model.NewDefaultDeployParams(),
				Services:      []model.Service{model.NewDefaultServiceConfig()},
			})
			if err != nil {
				fmt.Printf("Error deploying flycd in monitoring mode: %v\n", err)
				os.Exit(1)
			}
		},
	}

	flags.Init(cmd)

	return cmd
}
