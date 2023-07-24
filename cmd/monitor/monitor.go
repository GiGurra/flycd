package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/ext/fly_client"
	"github.com/gigurra/flycd/pkg/ext/github"
	"github.com/gigurra/flycd/pkg/util/util_cobra"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type flags struct {
	whIfc       *string
	whPath      *string
	whPort      *int
	startupSync *bool
}

func (f *flags) Init(cmd *cobra.Command) {
	f.whIfc = cmd.Flags().StringP("webhook-interface", "i", os.Getenv("WEBHOOK_INTERFACE"), "Webhook interface")
	f.whPath = cmd.Flags().StringP("webhook-path", "w", os.Getenv("WEBHOOK_PATH"), "Webhook path")
	f.whPort = cmd.Flags().IntP("webhook-port", "p", defaultWhPort(), "Webhook port")
	f.startupSync = cmd.Flags().BoolP("sync-on-startup", "s", false, "Sync all apps on startup")
}

func defaultWhPort() int {

	portStr := os.Getenv("WEBHOOK_PORT")
	if portStr == "" {
		return 80
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Errorf("invalid webhook port (not a valid integer): '%s', %w", portStr, err))
	}

	return port
}

func handleShutdown(shutdownHandler func()) {
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(
			sig,
			// Possible fly.io shutdown signals
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGUSR1,
			syscall.SIGUSR2,
			syscall.SIGKILL,
			syscall.SIGSTOP,
		)
		select {
		case s := <-sig:
			fmt.Printf("Received signal: %s\n", s.String())
			shutdownHandler()
		}
	}()
}

func Cmd(
	ctx context.Context,
	flyClient fly_client.FlyClient,
	deployService domain.DeployService,
	webhookService domain.WebHookService,
) *cobra.Command {
	flags := flags{}
	return util_cobra.CreateCmd(&flags, func() *cobra.Command {
		return &cobra.Command{
			Use:   "monitor",
			Short: "(Used when installed in fly.io env) Monitors apps, listens to webhooks, grabs new states from git, etc",
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

				// ensure we have a token loaded for the org we are monitoring, by listing apps
				appstList, err := flyClient.ListApps(ctx)
				if err != nil {
					fmt.Printf("Error listing apps (do you have a valid fly.io token loaded?): %v\n", err)
					os.Exit(1)
				}

				fmt.Printf("Currently deployed apps: \n")
				for _, app := range appstList {
					fmt.Printf(" - name=%s, org=%s\n", app.Name, app.Org)
				}

				err = webhookService.Start(ctx)
				if err != nil {
					fmt.Printf("Error starting webhook service: %v\n", err)
					os.Exit(1)
				}

				if *flags.startupSync {
					fmt.Printf("Syncing/Deploying all apps in %s\n", path)

					deployCfg := model.
						NewDefaultDeployConfig().
						WithAbortOnFirstError(false)

					_, err := deployService.DeployAll(ctx, path, deployCfg)
					if err != nil {
						fmt.Printf("Error deploying: %v\n", err)
					}

				}

				// Install shutdown signal handler
				handleShutdown(func() {
					fmt.Printf("Placing 'shutdown-application' job at the end of the current local job queue\n")
					webhookService.EnqueueJob(func() {
						fmt.Printf("Reached end of job queue, shutting down!\n")
						os.Exit(0)
					})
					webhookService.CloseJobQueue()
				})

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
				e.POST(whPath, func(c echo.Context) error {
					return processWebhook(c, path, webhookService)
				})

				// Start server
				e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", *flags.whIfc, *flags.whPort)))
			},
		}
	})
}

// Handler
func processWebhook(c echo.Context, path string, webhookService domain.WebHookService) error {

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

	truncatedBodyStr := string(bodyBytes)
	// Truncate to max 512 bytes
	if len(truncatedBodyStr) > 512 {
		truncatedBodyStr = truncatedBodyStr[:512] + "..."
	}

	fmt.Printf("Received webhook: %s\n", truncatedBodyStr)

	// Try to deserialize as GitHub webhook payload
	var githubWebhookPayload github.PushWebhookPayload
	err = json.Unmarshal(bodyBytes, &githubWebhookPayload)
	if err != nil {
		fmt.Printf("ERROR: deserializing github webhook payload: %v\n", err)
		return c.String(http.StatusBadRequest, "Error deserializing webhook payload")
	}

	ch := webhookService.HandleGithubWebhook(githubWebhookPayload, path)
	// TODO: Probably busy processing... Fix later and hand over to persistent queue
	select {
	case result := <-ch:
		if result != nil {
			fmt.Printf("ERROR: handling github webhook: %v\n", result)
			return c.String(http.StatusInternalServerError, "something went wrong - check flycd server logs!")
		} else {
			return c.String(http.StatusAccepted, "Too fast... something could be wrong")
		}
	case <-time.After(1 * time.Second):
		return c.String(http.StatusAccepted, "This is probably ok ;). ")
	}

}

// Handler
func processHealth(c echo.Context) error {

	return c.String(http.StatusOK, "Hello, World!")
}
