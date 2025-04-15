//  BYZRA ⸻ cmd/main.go <>
// +-----------------------------------------------------------+
//  doooooo ,8b.     888       8888 888PPP8b   ,dbPPPp ,8b.    |
//  d88     88'8o    888       8888 d88    `   d88ooP' 88'8o   |
//  d88     88PPY8.  888       8888 d8b PPY8 ,88' P'   88PPY8. |____________________________________________
//  d888888 8b   `Y' 888PPPPP  8888 Y8PPPPPP 88  do    8b   `Y' .go <--| CLI entrypoint and command routing +

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"caligra/internal/analyse"
	"caligra/internal/daemon"
	"caligra/internal/util"
	"caligra/internal/wipe"
)

func main() {
	util.Wiper()

	printHeader()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "analyse", "analyze":
		handleAnalyseCommand(os.Args[2:])
	case "wipe":
		handleWipeCommand(os.Args[2:])
	case "daemon":
		handleDaemonCommand(os.Args[2:])
	case "help":
		util.Wiper()
		printUsage()
	case "version":
		printVersion()
	default:
		util.Wiper()
		fmt.Println(util.LBL.Render("[!] Unknown command: " + command))
		printUsage()
		os.Exit(1)
	}
}

func handleAnalyseCommand(args []string) {
	util.Wiper()

	if len(args) < 1 {
		fmt.Println(util.LBL.Render("[X] No file specified for analysis"))
		fmt.Println(util.SUB.Render("Usage: caligra analyse <file>"))
		os.Exit(1)
	}

	path := args[0]

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(util.LBL.Render("[X] File not found: " + path))
		os.Exit(1)
	}

	fmt.Println(util.NSH.Render("[~] Analyzing: " + path))

	result, err := util.SpinWhile("[~] Analyzing metadata", func() (string, error) {
		report, err := analyse.Analyze(path)
		if err != nil {
			return "", err
		}
		return analyse.GenerateReport(report), nil
	})

	if err != nil {
		fmt.Println(util.LBL.Render("[X] Analysis failed: " + err.Error()))
		os.Exit(1)
	}

	fmt.Println(util.LBL.Render("[✓] Analysis completed successfully"))
	fmt.Println(result)
}

func handleWipeCommand(args []string) {
	util.Wiper()

	if len(args) < 1 {
		fmt.Println(util.LBL.Render("[X] No file specified for wiping"))
		fmt.Println(util.SUB.Render("Usage: caligra wipe <file> [options]"))
		os.Exit(1)
	}

	path := args[0]

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(util.LBL.Render("[X] File not found: " + path))
		os.Exit(1)
	}

	options := wipe.DefaultWipeOptions()

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--no-profile":
			options.InjectProfile = false
		case "--in-place":
			options.CreateCopy = false
		case "--no-backup":
			options.KeepBackup = false
		case "--secure":
			options.SecureDelete = true
		}
	}

	fmt.Println(util.NSH.Render("[~] Processing: " + path))

	result, err := util.SpinWhile("[~] Removing metadata", func() (string, error) {
		result, err := wipe.WipeFile(path, options)
		if err != nil {
			return "", err
		}
		return wipe.FormatWipeResult(result), nil
	})

	if err != nil {
		fmt.Println(util.LBL.Render("[X] Wipe failed: " + err.Error()))
		os.Exit(1)
	}

	fmt.Println(util.LBL.Render("[✓] Wipe completed successfully"))
	fmt.Println(result)
}

func handleDaemonCommand(args []string) {
	util.Wiper()

	if len(args) < 1 {
		fmt.Println(util.LBL.Render("[X] Daemon mode requires a subcommand"))
		fmt.Println(util.SUB.Render("Usage: caligra daemon [on|off|status]"))
		os.Exit(1)
	}

	subcommand := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(util.LBL.Render("[X] Cannot determine home directory"))
		os.Exit(1)
	}

	pidFile := filepath.Join(homeDir, ".caligra", "daemon.pid")

	switch subcommand {
	case "on", "start":
		if isDaemonRunning(pidFile) {
			fmt.Println(util.NSH.Render("[!] Daemon is already running"))
			os.Exit(0)
		}

		fmt.Println(util.NSH.Render("[~] Starting daemon..."))

		d, err := daemon.NewDaemon("")
		if err != nil {
			fmt.Println(util.LBL.Render("[X] Failed to create daemon: " + err.Error()))
			os.Exit(1)
		}

		if err := d.Start(); err != nil {
			fmt.Println(util.LBL.Render("[X] Failed to start daemon: " + err.Error()))
			os.Exit(1)
		}

		pid := os.Getpid()
		if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
			fmt.Println(util.LBL.Render("[!] Could not create daemon directory"))
		}

		pidBytes := make([]byte, 0, 16) // pre-allocate reasonable capacity for pid
		pidBytes = fmt.Appendf(pidBytes, "%d", pid)
		if err := os.WriteFile(pidFile, pidBytes, 0644); err != nil {
			fmt.Println(util.LBL.Render("[!] Could not write PID file"))
		}

		fmt.Println(util.NSH.Render("[✓] Daemon started successfully"))

		// keep running until interrupted
		select {}

	case "off", "stop":
		if !isDaemonRunning(pidFile) {
			fmt.Println(util.NSH.Render("[!] Daemon is not running"))
			os.Exit(0)
		}

		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println(util.LBL.Render("[X] Could not read daemon PID"))
			os.Exit(1)
		}

		pidStr := strings.TrimSpace(string(pidBytes))
		fmt.Println(util.NSH.Render("[~] Stopping daemon (PID " + pidStr + ")..."))

		// Send signal to daemon process
		// In a real implementation, we might use IPC or signals
		// For this example, we just remove the PID file
		if err := os.Remove(pidFile); err != nil {
			fmt.Println(util.LBL.Render("[X] Could not remove PID file"))
			os.Exit(1)
		}

		fmt.Println(util.NSH.Render("[✓] Daemon stopped"))

	case "status":
		if isDaemonRunning(pidFile) {
			pidBytes, _ := os.ReadFile(pidFile)
			pidStr := strings.TrimSpace(string(pidBytes))
			fmt.Println(util.NSH.Render("[...] Daemon is running (PID " + pidStr + ")"))

			// In a real implementation, we would get more status info
			// This could include watched directories, processed files, etc.
		} else {
			fmt.Println(util.NSH.Render("[...] Daemon is not running"))
		}

	default:
		fmt.Println(util.LBL.Render("[X] Unknown daemon command: " + subcommand))
		fmt.Println(util.SUB.Render("Usage: caligra daemon [on|off|status]"))
		os.Exit(1)
	}
}

func isDaemonRunning(pidFile string) bool {
	_, err := os.Stat(pidFile)
	return err == nil
}

func printHeader() {
	const art = `
	doooooo ,8b.     888       8888 888PPP8b   ,dbPPPp ,8b.
	d88     88'8o    888       8888 d88    ´   d88ooP' 88'8o
	d88     88PPY8.  888       8888 d8b PPY8 ,88' P'   88PPY8.
	d888888 8b   ´Y' 888PPPPP  8888 Y8PPPPPP 88  do    8b   ´Y'
`

	fmt.Printf("\n%s\n", util.LBL.Render(art))
	fmt.Printf("%s %s\n\n",
		util.NSH.Render("	→"),
		util.SHE.Render("CLI Metadata Control Utility"))
}

func printUsage() {
	fmt.Println(util.LBL.Render("USAGE"))
	fmt.Println("  caligra <command> [options]")
	fmt.Println("")
	fmt.Println(util.LBL.Render("COMMANDS"))
	fmt.Println("  analyse <file>          analyze metadata in a file")
	fmt.Println("  wipe <file> [options]   remove metadata from a file")
	fmt.Println("  daemon <on|off|status>  manage background monitoring service")
	fmt.Println("  help                    show this help information")
	fmt.Println("  version                 show version information")
	fmt.Println("")
	fmt.Println(util.LBL.Render("WIPE OPTIONS"))
	fmt.Println("  --no-profile            don't inject profile metadata")
	fmt.Println("  --in-place              modify file in place (don't create copy)")
	fmt.Println("  --no-backup             don't keep backup of original file")
	fmt.Println("  --secure                securely overwrite original data")
}

func printVersion() {
	util.Wiper()

	fmt.Println(util.LBL.Render("CALIGRA v1.0.0"))
	fmt.Println(util.LBL.Render("→ A CLI metadata control utility for Linux"))
	fmt.Println("")
	fmt.Println(util.NSH.Render("Copyright (c) 2025 bxavaby"))
	fmt.Println(util.SHE.Render("https://github.com/bxavaby/caligra"))
}
