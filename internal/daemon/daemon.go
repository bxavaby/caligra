// BYZRA ⸻ internal/daemon/daemon.go
// daemon management for background processing

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"caligra/internal/analyse"
	"caligra/internal/config"
	"caligra/internal/wipe"
)

// background service that monitors files
type Daemon struct {
	config  *config.DaemonConfig
	logger  *Logger
	watcher *Watcher
	running bool
}

// current state of the daemon
type DaemonStatus struct {
	Running        bool
	WatchedDirs    []string
	FileTypes      []string
	ProcessedFiles int
	ErrorCount     int
	StartTime      time.Time
}

// new daemon instance
func NewDaemon(configPath string) (*Daemon, error) {
	cfg, err := config.LoadDaemonConfig()
	if err != nil {
		cfg = config.GetDefaultConfig()
	}

	logDir := filepath.Join(os.Getenv("HOME"), ".caligra/logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger, err := NewLogger(filepath.Join(logDir, "caligra-daemon.log"), LevelInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	daemon := &Daemon{
		config: cfg,
		logger: logger,
	}

	return daemon, nil
}

func (d *Daemon) Start() error {
	if d.running {
		return fmt.Errorf("daemon already running")
	}

	d.logger.Info("Starting daemon")

	options := WatchOptions{
		Extensions:  d.config.Filter.Extensions,
		ExcludeDirs: []string{".git", "node_modules", ".venv"},
		MinFileAge:  2 * time.Second,
		Recursive:   true,
	}

	fileHandler := func(path string) error {
		// analyze file
		report, err := analyse.Analyze(path)
		if err != nil {
			d.logger.Warning(fmt.Sprintf("[!] Analysis failed for %s: %v", path, err))
			return err
		}

		// no sensitive metadata = no need to wipe
		if len(report.SensitiveFields) == 0 {
			d.logger.Debug(fmt.Sprintf("No sensitive metadata in %s, skipping", path))
			return nil
		}

		// sensitive metadata found = perform wipe
		d.logger.Info(fmt.Sprintf("Found %d sensitive fields in %s, wiping",
			len(report.SensitiveFields), path))

		// wiping options
		wipeOptions := &wipe.WipeOptions{
			InjectProfile: true,
			CustomProfile: nil, // default profile
			CreateCopy:    true,
			KeepBackup:    true,
			SecureDelete:  false,
		}

		// perform wipe
		result, err := wipe.WipeFile(path, wipeOptions)
		if err != nil {
			d.logger.Error(fmt.Sprintf("[X] Wipe failed for %s: %v", path, err))
			return err
		}

		if result.Success {
			d.logger.Info(fmt.Sprintf("Successfully processed %s → %s",
				path, result.OutputPath))
		} else {
			d.logger.Warning(fmt.Sprintf("[!] Wipe completed with issues for %s: %v",
				path, result.WipeErrors))
		}

		return nil
	}

	// create and start watcher
	watcher, err := NewWatcher(d.config.Watch.Paths, options, fileHandler, d.logger)
	if err != nil {
		d.logger.Error(fmt.Sprintf("[X] Failed to create watcher: %v", err))
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := watcher.Start(); err != nil {
		d.logger.Error(fmt.Sprintf("[X] Failed to start watcher: %v", err))
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	d.watcher = watcher
	d.running = true
	d.logger.Info("Daemon started successfully")

	return nil
}

// halts the daemon
func (d *Daemon) Stop() error {
	if !d.running {
		return nil
	}

	d.logger.Info("Stopping daemon")

	// stop watcher
	if d.watcher != nil {
		if err := d.watcher.Stop(); err != nil {
			d.logger.Warning(fmt.Sprintf("[!] Error stopping watcher: %v", err))
		}
	}

	// close logger
	if err := d.logger.Close(); err != nil {
		return fmt.Errorf("error closing logger: %w", err)
	}

	d.running = false
	return nil
}

// current daemon status
func (d *Daemon) Status() *DaemonStatus {
	if !d.running {
		return &DaemonStatus{
			Running: false,
		}
	}

	return &DaemonStatus{
		Running:     true,
		WatchedDirs: d.config.Watch.Paths,
		FileTypes:   d.config.Filter.Extensions,
	}
}

// is daemon currently running?
func (d *Daemon) IsRunning() bool {
	return d.running
}
