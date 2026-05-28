package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/version"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// releaseChecker represents the release checker
//
// Once created, call rc.run() to check for the latest release
type releaseChecker struct {
	logger        *logger.Logger
	latestRelease string
	mu            sync.RWMutex
	httpClient    *http.Client
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// githubRelease represents the GitHub response
type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetLatestRelease returns the latest release version
func (rc *releaseChecker) GetLatestRelease() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.latestRelease
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// run checks GitHub for the latest release
func (rc *releaseChecker) run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/geerew/offcourse/releases/latest", nil)
	if err != nil {
		rc.logger.Error().Err(err).Msg("Failed to create release check request")
		return err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "offcourse-release-checker")

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		rc.logger.Error().Err(err).Msg("Failed to check for latest release")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rc.logger.Warn().
			Int("status_code", resp.StatusCode).
			Msg("Failed to fetch latest release from GitHub")
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		rc.logger.Error().Err(err).Msg("Failed to decode GitHub release response")
		return err
	}

	rc.mu.Lock()
	oldRelease := rc.latestRelease
	rc.latestRelease = release.TagName
	rc.mu.Unlock()

	if oldRelease != release.TagName {
		rc.logger.Info().
			Str("latest_release", release.TagName).
			Str("current_version", version.GetVersion()).
			Msg("Latest release updated")
	}

	return nil
}
