package monitor

import (
	"context"
	"flycd/internal/flycd"
	"flycd/internal/flycd/util/util_cmd"
	"flycd/internal/flycd/util/util_tab_table"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"strings"
)

var flags struct {
	whPath      *string
	whPort      *int
	startupSync *bool
}

var Cmd = &cobra.Command{
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

		ctx := context.Background()

		// Get access token from env var
		accessToken := os.Getenv("FLY_ACCESS_TOKEN")
		if accessToken == "" {
			fmt.Printf("WARNING: FLY_ACCESS_TOKEN env var not set. Proceeding and assuming you are running locally logged in...\n")
		} else {
			ctx = context.WithValue(ctx, "FLY_ACCESS_TOKEN", accessToken)
		}

		// Get flycd ssh key from env var
		fmt.Printf("Checking if to store ssh... \n")
		sshKey := os.Getenv("FLY_SSH_PRIVATE_KEY")
		sshKeyName := os.Getenv("FLY_SSH_PRIVATE_KEY_NAME")
		if sshKey == "" {
			fmt.Printf("WARNING: FLY_SSH_PRIVATE_KEY env var not set. Proceeding and assuming you only want to access public repos, or you have magically solved git auth in some other way...\n")
		} else {

			fmt.Printf("FLY_SSH_PRIVATE_KEY env var is set, so we probably want o do something... \n")

			if sshKeyName == "" {
				fmt.Printf("FLY_SSH_PRIVATE_KEY_NAME env var not set, so just guessing we want 'id_rsa'\n")
				sshKeyName = "id_rsa"
			}

			fmt.Printf("Checking if to store ssh key: %s\n", sshKeyName)

			// Check that we
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("Error getting user home directory: %v\n", err)
				os.Exit(1)
			}

			sshDir := homeDir + "/.ssh"
			sshKeyPath := sshDir + "/" + sshKeyName

			// Ensure ssh dir exists
			if _, err := os.Stat(sshDir); os.IsNotExist(err) {
				fmt.Printf("ssh dir does not exist: %s\n", sshDir)
				os.Exit(1)
			}

			// Don't overwrite existing key
			if _, err := os.Stat(sshKeyPath); !os.IsNotExist(err) {
				fmt.Printf("ssh key already exists. Skipping copy from env var -> %s\n", sshKeyPath)
			} else {

				// Write key to file
				err = os.WriteFile(sshKeyPath, []byte(sshKey), 0600)
				if err != nil {
					fmt.Printf("Error writing ssh key to file: %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Stored ssh key: %s\n", sshKeyPath)
			}
		}

		// ensure we have a token loaded for the org we are monitoring
		res, err := util_cmd.NewCommand("flyctl", "apps", "list").Run(ctx)
		if err != nil {
			fmt.Printf("Error getting apps list. Do you have a token loaded?: %v\n", err)
			os.Exit(1)
		}

		appsTable, err := util_tab_table.ParseTable(res.StdOut)
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

		if *flags.startupSync {
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

		whPath := *flags.whPath
		if whPath == "" {
			whPath = "/webhook"
		}
		if !strings.HasPrefix(whPath, "/") {
			whPath = "/" + whPath
		}

		fmt.Printf("Listening on webhook path: %s\n", whPath)
		fmt.Printf("Listening on webhook port: %d\n", *flags.whPort)

		// Routes
		e.GET("/", processHealth)
		e.POST(whPath, processWebhook)

		// Start server
		e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", *flags.whPort)))
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

// Handler
func processHealth(c echo.Context) error {

	return c.String(http.StatusOK, "Hello, World!")
}

func init() {
	flags.whPath = Cmd.Flags().StringP("webhook-path", "w", os.Getenv("WEBHOOK_PATH"), "Webhook path")
	flags.whPort = Cmd.Flags().IntP("webhook-port", "p", 80, "Webhook port")
	flags.startupSync = Cmd.Flags().BoolP("sync-on-startup", "s", false, "Sync all apps on startup")
}
