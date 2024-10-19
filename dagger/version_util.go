package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
)

// determineVersions will return the latest version and the new version
func determineVersions(ctx context.Context, requestedVersion string) (string, string, error) {
	latestVersion, err := getLatestVersion(ctx)
	if err != nil {
		return "", "", err
	}

	parsedLatestVersion, err := semver.NewVersion(latestVersion)
	if err != nil {
		return "", "", err
	}

	switch requestedVersion {
	case "major":
		return latestVersion, fmt.Sprintf("%d.0.0", parsedLatestVersion.Major()+1), nil
	case "minor":
		return latestVersion, fmt.Sprintf("%d.%d.0", parsedLatestVersion.Major(), parsedLatestVersion.Minor()+1), nil
	case "patch":
		return latestVersion, fmt.Sprintf("%d.%d.%d", parsedLatestVersion.Major(), parsedLatestVersion.Minor(), parsedLatestVersion.Patch()+1), nil
	default:
		return latestVersion, requestedVersion, nil
	}
}

func getLatestVersion(ctx context.Context) (string, error) {
	resp, err := http.DefaultClient.Get("https://api.github.com/repos/queryplan-ai/queryplan-proxy/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}
