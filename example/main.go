package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/zeitlos/knockknock"
	"github.com/zeitlos/knockknock/config"
)

// Will be overwritten in production builds
var Version = "0.0.0"

const (
	AppName = "testapp"
)

func main() {
	knockknock.Run(config.New("testapp").WithRepo("ghcr.io/zeitlos/knockknock/testapp").WithVersion(Version), run)
}

func run() {
	port := flag.String("port", "8080", "port to listen on")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s v%s\n", AppName, Version)
		os.Exit(0)
	}

	// Setup routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/update", handleUpdate)
	http.HandleFunc("/rollback", handleRollback)

	// Start server
	addr := ":" + *port
	log.Printf("%s v%s starting on %s", AppName, Version, addr)
	log.Printf("Health endpoint: http://localhost%s/health", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	update, versions, err := knockknock.Client().CheckForUpdate(r.Context())

	if err != nil {
		slog.Error("failed to check for update from knockknock", "error", err)
	}

	history, err := knockknock.Client().History(r.Context())

	if err != nil {
		slog.Error("failed to get history from knockknock", "error", err)
	}

	versionOptions := ""
	for _, v := range versions {
		selected := ""
		current := ""
		if v.String() == Version {
			selected = "selected"
			current = " (current)"
		}
		versionOptions += fmt.Sprintf(`<option value="%s" %s>%s%s</option>`, v, selected, v, current)
	}

	historyHTML := ""
	for _, entry := range history {
		historyHTML += fmt.Sprintf("<li>%s <br />%s</li>", entry.Version, entry.LastInstalled.Format(time.DateTime))
	}

	newVersionClass := ""
	newVersionStyle := "display: none;"
	if update != nil {
		newVersionClass = "new-version-available"
		newVersionStyle = "display: inline-block;"
	}

	rollbackDisabled := ""
	rollbackOpacity := "1"
	if len(history) < 1 {
		rollbackDisabled = "disabled"
		rollbackOpacity = "0.5"
	}

	bgGradient := versionToColor(Version)

	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s v%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: %s;  /* <-- Use generated gradient */
            color: white;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            padding: 40px;
            border-radius: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        h1 { margin-top: 0; font-size: 3em; }
        .version {
            display: inline-block;
            background: rgba(255, 255, 255, 0.2);
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 0.5em;
            vertical-align: middle;
        }
        .info {
            background: rgba(0, 0, 0, 0.2);
            padding: 20px;
            border-radius: 10px;
            margin-top: 20px;
        }
        a {
            color: #ffd700;
            text-decoration: none;
            font-weight: bold;
        }
        a:hover { text-decoration: underline; }
        .status { font-size: 1.5em; }

        /* Update controls */
        .update-section {
            background: rgba(0, 0, 0, 0.3);
            padding: 25px;
            border-radius: 15px;
            margin-top: 20px;
            border: 2px solid rgba(255, 255, 255, 0.2);
        }
        .update-section.new-version-available {
            background: linear-gradient(135deg, rgba(255, 215, 0, 0.3) 0%%, rgba(255, 140, 0, 0.3) 100%%);
            border: 3px solid #ffd700;
            animation: pulse 2s ease-in-out infinite;
        }
        @keyframes pulse {
            0%%, 100%% { box-shadow: 0 0 0 0 rgba(255, 215, 0, 0.7); }
            50%% { box-shadow: 0 0 20px 10px rgba(255, 215, 0, 0); }
        }
        .version-selector {
            display: flex;
            gap: 10px;
            align-items: center;
            margin-top: 15px;
        }
        select {
            flex: 1;
            padding: 12px 16px;
            font-size: 16px;
            border: 2px solid rgba(255, 255, 255, 0.3);
            border-radius: 10px;
            background: rgba(0, 0, 0, 0.3);
            color: white;
            cursor: pointer;
            font-family: monospace;
        }
        select:focus {
            outline: none;
            border-color: #ffd700;
        }
        option {
            background: #2d2d2d;
            color: white;
        }
        .update-button {
            padding: 12px 30px;
            font-size: 16px;
            font-weight: bold;
            border: none;
            border-radius: 10px;
            background: linear-gradient(135deg, #ffd700 0%%, #ffaa00 100%%);
            color: #000;
            cursor: pointer;
            transition: all 0.3s ease;
            white-space: nowrap;
        }
        .update-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(255, 215, 0, 0.4);
        }
        .update-button:active {
            transform: translateY(0);
        }
        .new-version-badge {
            background: #ff4444;
            color: white;
            padding: 8px 15px;
            border-radius: 20px;
            font-weight: bold;
            font-size: 14px;
            animation: bounce 1s ease-in-out infinite;
            margin-bottom: 10px;
        }

        .rollback-button {
            padding: 12px 30px;
            font-size: 16px;
            font-weight: bold;
            border: none;
            border-radius: 10px;
            background: linear-gradient(135deg, #ff6b6b 0%%, #ee5a52 100%%);
            color: white;
            cursor: pointer;
            transition: all 0.3s ease;
            white-space: nowrap;
            margin-top: 15px;
        }
        .rollback-button:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(255, 107, 107, 0.4);
        }
        .rollback-button:active:not(:disabled) {
            transform: translateY(0);
        }
        .rollback-button:disabled {
            cursor: not-allowed;
            opacity: 0.5;
        }

        .history-list {
            list-style: none;
            padding: 0;
        }
        .history-list li {
            background: rgba(0, 0, 0, 0.2);
            padding: 8px 15px;
            margin: 5px 0;
            border-radius: 8px;
            font-family: monospace;
        }
        .info ol {
            padding-left: 30px;
            margin: 15px 0;
        }
        .info ol li {
            background: rgba(255, 255, 255, 0.15);
            padding: 12px 20px;
            margin: 8px 0;
            border-radius: 10px;
            font-family: monospace;
            font-size: 1.1em;
            border-left: 4px solid rgba(255, 215, 0, 0.6);
            transition: all 0.3s ease;
        }
        .info ol li:hover {
            background: rgba(255, 255, 255, 0.25);
            transform: translateX(5px);
        }
        @keyframes bounce {
            0%%, 100%% { transform: translateY(0); }
            50%% { transform: translateY(-5px); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s <span class="version">v%s</span></h1>

        <p class="status">Application is running</p>

        <div class="update-section %s">
            <span class="new-version-badge" style="%s">NEW VERSION AVAILABLE</span>
            <h3>Version Management</h3>
            <p><strong>Current Version:</strong> %s</p>

            <form method="POST" action="/update" class="version-selector">
                <select name="version" required>
                    %s
                </select>
                <button type="submit" class="update-button">
                    Update
                </button>
            </form>
        </div>

        <div class="info">
            <h3>Version History</h3>
            <ol>
                %s
            </ol>

            <form method="POST" action="/rollback">
                <button type="submit" class="rollback-button" %s style="opacity: %s"
                        onclick="return confirm('Are you sure you want to rollback to the previous version?')">
                    Rollback
                </button>
            </form>
        </div>

        <div class="info">
            <p><strong>Started:</strong> %s</p>
            <p><strong>Uptime:</strong> %s</p>
        </div>
    </div>
</body>
</html>
`, AppName, Version, bgGradient, AppName, Version, newVersionClass, newVersionStyle, Version, versionOptions, historyHTML, rollbackDisabled, rollbackOpacity, startTime.Format(time.RFC3339), time.Since(startTime).Round(time.Second))

	w.Write([]byte(html))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":  "healthy",
		"version": Version,
		"app":     AppName,
		"time":    time.Now().Format(time.RFC3339),
		"uptime":  time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}

// handleUpdate processes the update form submission
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	selectedVersion := r.FormValue("version")
	if selectedVersion == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	if selectedVersion == Version {
		// Redirect back with no change
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	log.Printf("Update requested to version: %s", selectedVersion)

	bgGradient := versionToColor(Version)

	// Show a success page before restart
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Updating %s</title>
    <meta http-equiv="refresh" content="15;url=/">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 600px;
            margin: 100px auto;
            padding: 40px;
            background: %s;
            color: white;
            text-align: center;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            padding: 60px;
            border-radius: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        h1 { font-size: 2.5em; margin-bottom: 20px; }
        .spinner {
            width: 60px;
            height: 60px;
            border: 6px solid rgba(255, 255, 255, 0.3);
            border-top-color: #ffd700;
            border-radius: 50%%;
            animation: spin 1s linear infinite;
            margin: 30px auto;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        p { font-size: 1.2em; line-height: 1.6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Updating...</h1>
        <div class="spinner"></div>
        <p><strong>Updating to version %s</strong></p>
        <p>The application will restart shortly.</p>
        <p>This page will automatically refresh in a few seconds.</p>
    </div>
</body>
</html>
`, AppName, bgGradient, selectedVersion)

	w.Write([]byte(html))

	if err := knockknock.Client().Update(context.Background(), selectedVersion); err != nil {
		slog.Error("failed to update", "error", err)
	}
}

// handleRollback processes the rollback form submission
func handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	log.Printf("Rollback requested")

	bgGradient := versionToColor(Version)

	// Show a success page before restart
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Rolling Back %s</title>
    <meta http-equiv="refresh" content="15;url=/">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 600px;
            margin: 100px auto;
            padding: 40px;
            background: %s;
            color: white;
            text-align: center;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            padding: 60px;
            border-radius: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        h1 { font-size: 2.5em; margin-bottom: 20px; }
        .spinner {
            width: 60px;
            height: 60px;
            border: 6px solid rgba(255, 255, 255, 0.3);
            border-top-color: #ff6b6b;
            border-radius: 50%%;
            animation: spin 1s linear infinite;
            margin: 30px auto;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        p { font-size: 1.2em; line-height: 1.6; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Rolling Back...</h1>
        <div class="spinner"></div>
        <p><strong>Rolling back to previous version</strong></p>
        <p>The application will restart shortly.</p>
        <p>This page will automatically refresh in a few seconds.</p>
    </div>
</body>
</html>
`, AppName, bgGradient)

	w.Write([]byte(html))

	if err := knockknock.Client().Rollback(context.Background()); err != nil {
		slog.Error("failed to rollback", "error", err)
	}
}

func versionToColor(version string) string {
	// FNV-1a hash for better distribution
	hash := uint32(2166136261)
	for _, c := range version {
		hash ^= uint32(c)
		hash *= 16777619
	}

	// Additional mixing to improve avalanche effect
	hash ^= hash >> 16
	hash *= 0x85ebca6b
	hash ^= hash >> 13
	hash *= 0xc2b2ae35
	hash ^= hash >> 16

	hue1 := int(hash % 360)
	hue2 := (hue1 + 60) % 360

	color1 := fmt.Sprintf("hsl(%d, 70%%, 55%%)", hue1)
	color2 := fmt.Sprintf("hsl(%d, 70%%, 45%%)", hue2)

	return fmt.Sprintf("linear-gradient(135deg, %s 0%%, %s 100%%)", color1, color2)
}

var startTime = time.Now()
