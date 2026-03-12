package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/paths"
)

type Settings struct {
	BypassPermissions bool `json:"bypass_permissions"`
}

func globalSettingsPath() string {
	return filepath.Join(paths.DataDir(), "settings.json")
}

func LoadGlobalSettings() Settings {
	data, err := os.ReadFile(globalSettingsPath())
	if err != nil {
		return Settings{BypassPermissions: true} // default: bypass
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{BypassPermissions: true}
	}
	return s
}

func SaveGlobalSettings(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(globalSettingsPath(), append(data, '\n'), 0644)
}
