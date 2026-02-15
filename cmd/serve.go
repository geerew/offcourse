package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/geerew/off-course/api"
	"github.com/geerew/off-course/app"
	"github.com/geerew/off-course/utils/coursescan"
	"github.com/geerew/off-course/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// serveCmd starts the application
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the application",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		httpAddr := viper.GetString("http")
		dataDir := viper.GetString("data-dir")
		enableSignup := viper.GetBool("enable-signup")
		isDev := viper.GetBool("dev")
		debug := viper.GetBool("debug")

		appMode := app.AppModeProd
		if isDev {
			appMode = app.AppModeDev
		}

		appConfig := &app.Config{
			HttpAddr:     httpAddr,
			DataDir:      dataDir,
			AppMode:      appMode,
			EnableSignup: enableSignup,
			Debug:        debug,
		}

		application, err := app.NewApp(ctx, appConfig)
		if err != nil {
			os.Stderr.WriteString("Failed to initialize app: " + err.Error() + "\n")
			os.Exit(1)
		}

		appLogger := application.Logger.WithApp()

		appLogger.Info().
			Str("version", version.GetVersion()).
			Str("commit", version.GetCommit()).
			Msg("Starting OffCourse")

		go application.CourseScan.Worker(ctx, coursescan.Processor)

		application.Cron.Start()

		// Router
		router := api.NewRouter(application)

		var wg sync.WaitGroup
		wg.Add(1)

		// Listen for shutdown signals
		go func() {
			defer wg.Done()
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			<-quit
		}()

		go func() {
			defer wg.Done()
			if err := router.Serve(); err != nil {
				appLogger.Error().Err(err).Msg("Failed to start router")
				os.Exit(1)
			}
		}()

		wg.Wait()

		appLogger.Info().Msg("Shutting down...")

		if err := application.Close(); err != nil {
			appLogger.Error().Err(err).Msg("Failed to close application resources")
		}

	},
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolP("dev", "d", false, "Run in development mode")
	serveCmd.Flags().String("http", "127.0.0.1:9081", "TCP address to listen for the HTTP server")
	serveCmd.Flags().String("data-dir", "./oc_data", "Directory to store data files")
	serveCmd.Flags().Bool("enable-signup", false, "Allow users to create new accounts")
	serveCmd.Flags().Bool("debug", false, "Enable debug logging")

	// Bind flags
	viper.SetEnvPrefix("OC")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind each flag
	_ = viper.BindPFlag("dev", serveCmd.Flags().Lookup("dev"))
	_ = viper.BindPFlag("http", serveCmd.Flags().Lookup("http"))
	_ = viper.BindPFlag("data-dir", serveCmd.Flags().Lookup("data-dir"))
	_ = viper.BindPFlag("enable-signup", serveCmd.Flags().Lookup("enable-signup"))
	_ = viper.BindPFlag("debug", serveCmd.Flags().Lookup("debug"))
}
