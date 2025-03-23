package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

func watchChanges(ctx context.Context, dir string, callback func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				if err = watcher.Close(); err != nil {
					slog.Error("unable to close the fs watcher", "error", err)
				}

				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				slog.Debug("fs event", "event", event)

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					slog.Debug("file modified", "file", event.Name)
					callback()
				}

			case watchErr, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Debug("fs watch error:", "error", watchErr)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	return nil
}
