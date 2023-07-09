package main

import (
	"context"
	"flycd/internal/flycd"
	"flycd/internal/flycd/util/util_cmd"
	"flycd/internal/flycd/util/util_tab_table"
	"flycd/internal/flyctl"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const Version = "v0.0.9"

var rootCmd = &cobra.Command{
	Use:   "flycd",
	Short: "flycd deployment of fly apps entirely from code, without manual flyctl commands... I hope :D",
	Long:  `Complete documentation is available at https://github.com/gigurra/flycd`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			err := cmd.Help() // Display help message
			if err != nil {
				fmt.Printf("Error displaying help: %v\n", err)
			}
			err = cmd.Usage()
			if err != nil {
				fmt.Printf("error displaying usage: %v\n", err)
			}
			os.Exit(1) // Exit with code 1
		}
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Manually deploy a single flycd app, or all flycd apps inside a folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		fmt.Printf("Deploying from: %s\n", path)

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			fmt.Printf("Error getting force flag: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		err = flycd.Deploy(ctx, path, force)
		if err != nil {
			fmt.Printf("Error deploying from %s: %v\n:", path, err)
			os.Exit(1)
		}
	},
}

var _ any = deployCmd.Flags().BoolP("force", "f", false, "Force re-deploy even if there are no changes")

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "(Used when installed in fly.io env) Monitors flycd apps, listens to webhooks, grabs new states from git, etc",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Monitoring: %s\n", path)

		// Get access token from env var
		accessToken := os.Getenv("FLY_ACCESS_TOKEN")
		if accessToken == "" {
			fmt.Printf("FLY_ACCESS_TOKEN env var not set. Please set it to a valid fly.io access token\n")
			os.Exit(1)
		}

		// For now, store the access token in a global. This is ugly :S. but... it's what we got right now :S
		ctx := context.Background()
		ctx = context.WithValue(ctx, "FLY_ACCESS_TOKEN", accessToken)

		// ensure we have a token loaded for the org we are monitoring
		appsTableString, err := util_cmd.NewCommand("flyctl", "apps", "list").Run(ctx)
		if err != nil {
			fmt.Printf("Error getting apps list. Do you have a token loaded?: %v\n", err)
			os.Exit(1)
		}

		appsTable, err := util_tab_table.ParseTable(appsTableString)
		if err != nil {
			fmt.Printf("Error parsing apps list: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Currently deployed apps: \n")
		for _, appRow := range appsTable.RowMaps {
			name := appRow["NAME"]
			org := appRow["OWNER"]

			fmt.Printf(" - name=%s, org=%s\n", name, org)
		}

		syncAllAppsOnStartup, err := cmd.Flags().GetBool("sync-on-startup")
		if err != nil {
			fmt.Printf("Error getting sync-on-startup flag: %v\n", err)
			os.Exit(1)
		}

		if syncAllAppsOnStartup {
			fmt.Printf("Syncing/Deploying all apps in %s\n", path)

			err = flycd.Deploy(ctx, path, false)
			if err != nil {
				fmt.Printf("Error deploying from %s: %v\n:", path, err)
				os.Exit(1)
			}
		}

		// Echo instance
		e := echo.New()

		// Middleware
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())

		whPath, err := cmd.Flags().GetString("webhook-path")
		if err != nil {
			fmt.Printf("Error getting webhook-path flag: %v\n", err)
			os.Exit(1)
		}
		if whPath == "" {
			whPath = "/webhook"
		}
		if !strings.HasPrefix(whPath, "/") {
			whPath = "/" + whPath
		}

		whPort, err := cmd.Flags().GetInt("webhook-port")
		if err != nil {
			fmt.Printf("Error getting webhook-port flag: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Listening on webhook path: %s\n", whPath)
		fmt.Printf("Listening on webhook port: %d\n", whPort)

		// Routes
		e.POST(whPath, processWebhook)

		// Start server
		e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", whPort)))

		// TODO: Ensure we have ssh keys loaded for cloning git repos. If running on fly.io, we need to copy them from /mnt/somewhere -> ~/.ssh
	},
}

// Handler
func processWebhook(c echo.Context) error {

	body := c.Request().Body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return c.String(http.StatusUnsupportedMediaType, "Error reading request body")
	}
	defer func(body io.Closer) {
		err := body.Close()
		if err != nil {
			fmt.Printf("Error closing request body: %v\n", err)
		}
	}(body)

	fmt.Printf("Received webhook: %s\n", string(bodyBytes))
	// TODO: Do something useful here

	return c.String(http.StatusOK, "Hello, World!")
}

var _ any = monitorCmd.Flags().StringP("webhook-path", "w", os.Getenv("WEBHOOK_PATH"), "Webhook path")
var _ any = monitorCmd.Flags().IntP("webhook-port", "p", 80, "Webhook port")
var _ any = monitorCmd.Flags().BoolP("sync-on-startup", "s", false, "Sync all apps on startup")

var installCmd = &cobra.Command{
	Use:   "install <flycd app name> <fly.io org slug> <fly.io region>",
	Short: "Install flycd into your fly.io account, listening to webhooks from this cfg repo and your app repos",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {

		appName := args[0]

		orgSlug := args[1]

		region := args[2]

		fmt.Printf("Installing flycd with app name '%s' to org '%s'\n", appName, orgSlug)

		ctx := context.Background()

		fmt.Printf("Check if flycd app already exists\n")
		appExists, err := flycd.AppExists(ctx, appName)
		if err != nil {
			fmt.Printf("Error checking if app exists: %v\n", err)
			os.Exit(1)
		}

		if !appExists {

			fmt.Printf("Creating a dummy app '%s' to reserve the name\n", appName)
			err = flycd.DeployAppFromConfig(ctx, false, flycd.AppConfig{
				App:           appName,
				Org:           orgSlug,
				PrimaryRegion: region,
				Source:        flycd.NewInlineDockerFileSource("FROM nginx:latest"),
				LaunchParams:  flycd.NewDefaultLaunchParams(appName, orgSlug),
				Services:      []flycd.Service{flycd.NewDefaultServiceConfig()},
			})
			if err != nil {
				fmt.Printf("Error creating dummy app: %v\n", err)
				os.Exit(1)
			}
		}

		existsAccessTokenSecret, err := flyctl.ExistsSecret(ctx, flyctl.ExistsSecretCmd{
			AppName:    appName,
			SecretName: "FLY_ACCESS_TOKEN",
		})
		if err != nil {
			fmt.Printf("Error checking if access token secret exists: %v\n", err)
			os.Exit(1)
		}

		if !existsAccessTokenSecret {

			fmt.Printf("App name successfully reserved... creating access token for org '%s'\n", orgSlug)
			token, err := flyctl.CreateOrgToken(orgSlug)
			if err != nil {
				fmt.Printf("Error creating org token: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Token created.. storing it...\n")

			err = flyctl.StoreSecret(ctx, flyctl.StoreSecretCmd{
				AppName:     appName,
				SecretName:  "FLY_ACCESS_TOKEN",
				SecretValue: token,
			})

			if err != nil {
				fmt.Printf("Error storing token: %v\n", err)
				os.Exit(1)
			}

		}

		fmt.Printf("Deploying flycd in monitoring mode to fly.io\n")
		err = flycd.DeployAppFromConfig(ctx, false, flycd.AppConfig{
			App:           appName,
			Org:           orgSlug,
			PrimaryRegion: region,
			Source:        flycd.NewLocalFolderSource("."),
			LaunchParams:  flycd.NewDefaultLaunchParams(appName, orgSlug),
			Services:      []flycd.Service{flycd.NewDefaultServiceConfig()},
		})
		if err != nil {
			fmt.Printf("Error deploying flycd in monitoring mode: %v\n", err)
			os.Exit(1)
		}
		// TODO: Add ssh keys as secrets so we can pull from other git repos argocd style
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade your flycd installation in your fly.io account to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current working directory: %v\n", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			path = args[0]
		}

		fmt.Printf("Monitoring: %s\n", path)

		fmt.Printf("Not implemented yet, sorry :(\n")
		os.Exit(1)
	},
}

func main() {
	fmt.Printf("Starting FlyCD %s...\n", Version)

	// Check that required applications are installed
	requiredApps := []string{"flyctl", "git", "ssh", "yj", "cat", "cp", "shasum", "awk"}
	for _, app := range requiredApps {
		_, err := exec.LookPath(app)
		if err != nil {
			fmt.Printf("Error: required app '%s' not found in PATH\n", app)
			os.Exit(1)
		}
	}

	// prepare cli
	rootCmd.AddCommand(deployCmd, monitorCmd, installCmd, upgradeCmd)

	// run cli
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("FlyCD %s exiting normally, bye!\n", Version)
}
