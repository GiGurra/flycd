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
		e.GET("/", processHealth)
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

// Handler
func processHealth(c echo.Context) error {

	return c.String(http.StatusOK, "Hello, World!")
}

var _ any = Cmd.Flags().StringP("webhook-path", "w", os.Getenv("WEBHOOK_PATH"), "Webhook path")
var _ any = Cmd.Flags().IntP("webhook-port", "p", 80, "Webhook port")
var _ any = Cmd.Flags().BoolP("sync-on-startup", "s", false, "Sync all apps on startup")
