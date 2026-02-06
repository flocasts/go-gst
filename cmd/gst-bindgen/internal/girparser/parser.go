package girparser

import (
	"encoding/xml"
	"fmt"
	"os"
)

// ParseRepository parses a GIR XML file and returns the Repository.
func ParseRepository(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var repo Repository
	if err := xml.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", path, err)
	}

	return &repo, nil
}
