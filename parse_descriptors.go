package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func parseDescriptorFiles(fileExtensions []string, dir string) ([]*descriptorpb.FileDescriptorSet, error) {
	logger := slog.With("descriptors_dir", dir)

	logger.Debug("analyzing descriptors directory")
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("get descriptors directory info: %w", err)
	}

	logger.Debug("descriptors directory info", "name", stat.Name(), "size", stat.Size(), "is_dir", stat.IsDir())
	if !stat.IsDir() {
		return nil, fmt.Errorf("descriptors directory is not a directory")
	}

	logger.Debug("looking for descriptor files", "extensions", fileExtensions)

	descriptors := make([]*descriptorpb.FileDescriptorSet, 0)
	err = filepath.Walk(dir, func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !slices.Contains(fileExtensions, ext) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			if errors.Is(readErr, os.ErrPermission) {
				logger.Warn("descriptor file not accessible", "path", path, "error", readErr)
				return nil
			}

			return fmt.Errorf("read descriptor file '%s': %w", path, err)
		}

		logger.Debug("found descriptor file, parsing contents...", "path", path)
		var fileDescriptorSet descriptorpb.FileDescriptorSet
		if err = proto.Unmarshal(content, &fileDescriptorSet); err != nil {
			return fmt.Errorf("unmarshal descriptor file at path '%s': %w", path, err)
		}

		logger.Debug("successfully parsed descriptor file", "path", path, "pb_files_count", len(fileDescriptorSet.GetFile()))
		descriptors = append(descriptors, &fileDescriptorSet)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search for .pb files in dir '%s': %w", dir, err)
	}

	return descriptors, nil
}
