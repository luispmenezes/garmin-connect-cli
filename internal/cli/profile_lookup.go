package cli

import (
	"encoding/json"
	"errors"

	"github.com/luispmenezes/garmin-connect-cli/internal/auth"
	"github.com/luispmenezes/garmin-connect-cli/internal/garmin"
)

func getDisplayName(client *garmin.Client, token auth.OAuth2Token) (string, error) {
	data, err := client.GetJSON(token, garmin.ProfilePath())
	if err != nil {
		return "", err
	}
	var profile map[string]any
	if err := json.Unmarshal(data, &profile); err != nil {
		return "", err
	}
	if displayName, ok := profile["displayName"].(string); ok && displayName != "" {
		return displayName, nil
	}
	return "", errors.New("profile response did not include displayName")
}
