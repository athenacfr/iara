package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/paths"
)

// Metadata holds cw project metadata persisted in .cw/metadata.json.
type Metadata struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
}

// SaveMetadata validates and writes metadata to <projectDir>/.cw/metadata.json.
func SaveMetadata(projectDir string, raw string) error {
	var m Metadata
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if m.Title == "" {
		return fmt.Errorf("title is required")
	}
	if m.Description == "" {
		return fmt.Errorf("description is required")
	}
	if m.Instructions == "" {
		return fmt.Errorf("instructions is required")
	}

	cwDir := filepath.Join(projectDir, ".cw")
	if err := os.MkdirAll(cwDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(cwDir, "metadata.json"), data, 0644)
}

// LoadMetadata reads .cw/metadata.json for a project by name.
func LoadMetadata(name string) (*Metadata, error) {
	dir := filepath.Join(paths.ProjectsDir(), name)
	return LoadMetadataAt(dir)
}

// LoadMetadataAt reads .cw/metadata.json from a project directory.
func LoadMetadataAt(projectDir string) (*Metadata, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, ".cw", "metadata.json"))
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// HasMetadata checks if a project has .cw/metadata.json.
func HasMetadata(name string) bool {
	dir := filepath.Join(paths.ProjectsDir(), name)
	_, err := os.Stat(filepath.Join(dir, ".cw", "metadata.json"))
	return err == nil
}
