package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/default23/protofake/mapper"
)

func parseMappingFiles(dir string) ([]*mapper.Mapping, error) {
	logger := slog.With("mappings_dir", dir)

	logger.Debug("analyzing mappings directory")
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("mappings directory not found, skipping")
			return nil, nil
		}

		return nil, fmt.Errorf("get descriptors directory info: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("mappings directory is not a directory")
	}

	logger.Debug("looking for .json mapping files")

	mappings := make([]*mapper.Mapping, 0)
	err = filepath.Walk(dir, func(path string, info fs.FileInfo, _ error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if ext := filepath.Ext(path); ext != ".json" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read mapping file '%s': %w", path, err)
		}

		content = bytes.TrimSpace(content)
		if len(content) == 0 {
			logger.Debug("empty mapping file, skipping", "path", path)
			return nil
		}

		switch content[0] {
		case '[':
			var mm []*mapper.Mapping
			if err = json.Unmarshal(content, &mm); err != nil {
				return fmt.Errorf("unmarshal mapping from file '%s': %w", path, err)
			}
			for _, m := range mm {
				if err = m.IsValid(); err != nil {
					return fmt.Errorf("validate mapping '%s' from file '%s': %w", m.Endpoint, path, err)
				}

				if !strings.HasPrefix(m.Endpoint, "/") {
					m.Endpoint = "/" + m.Endpoint
				}
			}

			mappings = append(mappings, mm...)
		case '{':
			m := new(mapper.Mapping)
			if err = json.Unmarshal(content, m); err != nil {
				return fmt.Errorf("unmarshal mapping from file '%s': %w", path, err)
			}
			if err = m.IsValid(); err != nil {
				return fmt.Errorf("validate mappings from file '%s': %w", path, err)
			}

			if !strings.HasPrefix(m.Endpoint, "/") {
				m.Endpoint = "/" + m.Endpoint
			}

			mappings = append(mappings, m)
		default:
			return fmt.Errorf("mapping file '%s' does not contain a valid JSON content", path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, m := range mappings {
		if strings.TrimSpace(m.ID) == "" {
			m.ID = uuid.NewString()
		}
	}

	return mappings, nil
}
