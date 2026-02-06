package overrides

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses an override YAML file.
func LoadFile(path string) (*PackageOverrides, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading override file %s: %w", path, err)
	}

	var ovr PackageOverrides
	if err := yaml.Unmarshal(data, &ovr); err != nil {
		return nil, fmt.Errorf("parsing override file %s: %w", path, err)
	}

	return &ovr, nil
}
