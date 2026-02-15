package cmd

// TODO: Handle password reset when the application is not running (just update the db)

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/auth"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// adminResetPasswordCmd resets the password for an admin user
var adminResetPasswordCmd = &cobra.Command{
	Use:   "reset-password <username>",
	Short: "Reset password for an admin user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]

		fmt.Println()
		fmt.Println("🔐 Admin Password Reset")
		fmt.Println("=======================")
		fmt.Println()

		// Get configuration
		dataDir := viper.GetString("data-dir")
		httpAddr := viper.GetString("http")
		appFs := appfs.New(afero.NewOsFs())

		if err := verifyAdminUser(username, dataDir); err != nil {
			errorMessage("%s", err)
			os.Exit(1)
		}

		var password string
		for {
			password = questionPassword("New Password")
			if password != "" {
				break
			}
			errorMessage("Password cannot be empty")
		}

		for {
			confirmPassword := questionPassword("Confirm Password")
			if confirmPassword == password {
				break
			}
			errorMessage("Passwords do not match")
		}

		fmt.Println()

		if err := resetPasswordViaAPI(appFs, username, password, dataDir, httpAddr); err != nil {
			errorMessage("Failed to reset password: %s", err)
			os.Exit(1)
		}

		successMessage("✅ Password reset successfully for '%s'", username)
	},
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// verifyAdminUser ensures a user exists and is admin
func verifyAdminUser(username, dataDir string) error {
	ctx := context.Background()
	appFs := appfs.New(afero.NewOsFs())

	dbManagerConfig := &database.DatabaseManagerConfig{
		DataDir: dataDir,
		AppFs:   appFs,
		Testing: false,
	}

	dbManager, err := database.NewSQLiteManager(dbManagerConfig)
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}

	appDao := dao.New(dbManager.DataDb)

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: username})
	user, err := appDao.GetUser(ctx, dbOpts)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user '%s' not found", username)
		}

		return fmt.Errorf("failed to lookup user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user '%s' not found", username)
	}

	if user.Role != types.UserRoleAdmin {
		return fmt.Errorf("user '%s' is not an admin user", username)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// resetPasswordViaAPI generates a recovery token on disk, then makes an HTTP request to
// the running application to reset the password
func resetPasswordViaAPI(appFs *appfs.AppFs, username, password, dataDir, httpAddr string) error {
	recoveryToken, err := auth.GenerateRecoveryToken(appFs, username, password, dataDir)
	if err != nil {
		return fmt.Errorf("failed to generate recovery token: %w", err)
	}

	defer func() {
		auth.DeleteRecoveryToken(appFs, dataDir)
	}()

	requestBody := map[string]string{
		"token": recoveryToken.Token,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/admin/recovery", httpAddr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("application is not running or not accessible at %s: %w", httpAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("recovery request failed with status %d", resp.StatusCode)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func init() {
	adminCmd.AddCommand(adminResetPasswordCmd)

	adminResetPasswordCmd.Flags().String("http", "127.0.0.1:9081", "TCP address to listen for the HTTP server")
	adminResetPasswordCmd.Flags().String("data-dir", "./oc_data", "Directory to store data files")

	// Bind flags
	viper.SetEnvPrefix("OC")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind each flag
	_ = viper.BindPFlag("http", adminResetPasswordCmd.Flags().Lookup("http"))
	_ = viper.BindPFlag("data-dir", adminResetPasswordCmd.Flags().Lookup("data-dir"))
}
