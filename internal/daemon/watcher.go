// BYZRA ⸻ internal/daemon/watcher.go
// file system monitoring for the daemon

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// processes a detected file
type FileHandler func(path string) error

// configures the watcher behavior
type WatchOptions struct {
	// extensions to monitor
	Extensions []string

	// directories to exclude
	ExcludeDirs []string

	// min file age before processing (avoid processing incomplete files)
	MinFileAge time.Duration

	// process files recursively in subdirectories?
	Recursive bool
}

// monitors directories for file changes
type Watcher struct {
	watcher     *fsnotify.Watcher
	dirs        []string
	options     WatchOptions
	handler     FileHandler
	logger      *Logger
	processed   map[string]time.Time
	processLock sync.Mutex
	running     bool
}

// new file system watcher
func NewWatcher(dirs []string, options WatchOptions, handler FileHandler, logger *Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	var validDirs []string
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			logger.Warning(fmt.Sprintf("Skipping invalid directory %s: %v", dir, err))
			continue
		}

		if !info.IsDir() {
			logger.Warning(fmt.Sprintf("Skipping non-directory path %s", dir))
			continue
		}

		validDirs = append(validDirs, dir)
	}

	if len(validDirs) == 0 {
		return nil, fmt.Errorf("no valid directories to watch")
	}

	return &Watcher{
		watcher:   fsWatcher,
		dirs:      validDirs,
		options:   options,
		handler:   handler,
		logger:    logger,
		processed: make(map[string]time.Time),
	}, nil
}

// begins watching the configured directories
func (w *Watcher) Start() error {
	if w.running {
		return fmt.Errorf("watcher already running")
	}

	// add directories to watch
	for _, dir := range w.dirs {
		if w.options.Recursive {
			if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					w.logger.Warning(fmt.Sprintf("Error accessing path %s: %v", path, err))
					return nil // Continue walking
				}

				if info.IsDir() {
					for _, exclude := range w.options.ExcludeDirs {
						if strings.Contains(path, exclude) {
							return filepath.SkipDir
						}
					}

					if err := w.watcher.Add(path); err != nil {
						w.logger.Warning(fmt.Sprintf("Failed to watch directory %s: %v", path, err))
					} else {
						w.logger.Debug(fmt.Sprintf("Watching directory: %s", path))
					}
				}
				return nil
			}); err != nil {
				w.logger.Error(fmt.Sprintf("Error walking directory %s: %v", dir, err))
			}
		} else {
			// Just watch the top-level directory
			if err := w.watcher.Add(dir); err != nil {
				w.logger.Warning(fmt.Sprintf("Failed to watch directory %s: %v", dir, err))
			} else {
				w.logger.Debug(fmt.Sprintf("Watching directory: %s", dir))
			}
		}
	}

	// start processing events
	go w.processEvents()

	// start cleanup routine
	go w.periodicCleanup()

	w.running = true
	w.logger.Info("File watcher started")

	return nil
}

// terminates the watcher
func (w *Watcher) Stop() error {
	if !w.running {
		return nil
	}

	err := w.watcher.Close()
	w.running = false
	w.logger.Info("File watcher stopped")

	return err
}

// checks if a file should be processed based on options
func (w *Watcher) shouldProcessFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if len(w.options.Extensions) > 0 {
		matched := false
		for _, allowedExt := range w.options.Extensions {
			if ext == strings.ToLower(allowedExt) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// is file old enough?
	if w.options.MinFileAge > 0 {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}

		// if file was modified less than MinFileAge ago, don't process
		if time.Since(info.ModTime()) < w.options.MinFileAge {
			return false
		}
	}

	w.processLock.Lock()
	defer w.processLock.Unlock()

	if lastProcessed, exists := w.processed[path]; exists {
		if time.Since(lastProcessed) < time.Minute {
			return false
		}
	}

	return true
}

func (w *Watcher) markProcessed(path string) {
	w.processLock.Lock()
	defer w.processLock.Unlock()

	w.processed[path] = time.Now()
}

// file system events
func (w *Watcher) processEvents() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return // Watcher was closed
			}

			// creation or write event?
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				path := event.Name

				// if a new directory was created and we're in recursive mode, watch it
				if w.options.Recursive {
					info, err := os.Stat(path)
					if err == nil && info.IsDir() {
						excluded := false
						for _, exclude := range w.options.ExcludeDirs {
							if strings.Contains(path, exclude) {
								excluded = true
								break
							}
						}

						if !excluded {
							if err := w.watcher.Add(path); err != nil {
								w.logger.Warning(fmt.Sprintf("[!] Failed to watch new directory %s: %v", path, err))
							} else {
								w.logger.Debug(fmt.Sprintf("Watching new directory: %s", path))
							}
						}
						continue
					}
				}

				if w.shouldProcessFile(path) {
					go func(filePath string) {
						// small delay to ensure file is completely written
						time.Sleep(500 * time.Millisecond)

						w.logger.Debug(fmt.Sprintf("Processing file: %s", filePath))

						if err := w.handler(filePath); err != nil {
							w.logger.Error(fmt.Sprintf("[X] Failed to process file %s: %v", filePath, err))
						} else {
							w.logger.Info(fmt.Sprintf("Successfully processed file: %s", filePath))
						}

						w.markProcessed(filePath)
					}(path)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return // watcher closed
			}
			w.logger.Error(fmt.Sprintf("[X] Watcher error: %v", err))
		}
	}
}

// periodically cleans the processed files map
func (w *Watcher) periodicCleanup() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processLock.Lock()

			// clean entries older than 1 hour
			cutoff := time.Now().Add(-1 * time.Hour)
			for path, processed := range w.processed {
				if processed.Before(cutoff) {
					delete(w.processed, path)
				}
			}

			w.processLock.Unlock()

			w.logger.Debug("Cleaned processed files cache")

		default:
			if !w.running {
				return
			}
			time.Sleep(1 * time.Second)
		}
	}
}
