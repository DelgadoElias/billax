package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// serveUI creates an HTTP handler for the SPA
// It serves static assets from the ui/dist folder and redirects all other requests to index.html
// This uses the filesystem instead of go:embed to support development workflows
// Production builds use Docker to bundle the dist folder into the binary
func serveUI() http.Handler {
	// Try to find ui/dist - could be at ./ui/dist or ../ui/dist depending on where the binary is run from
	var distPath string
	candidates := []string{
		"./ui/dist",
		"../ui/dist",
		"/app/ui/dist", // Docker path
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			distPath = candidate
			break
		}
	}

	// If dist folder not found, return 404 handler
	if distPath == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "UI not available (checked: ./ui/dist, ../ui/dist, /app/ui/dist)", http.StatusNotFound)
		})
	}


	fileServer := http.FileServer(http.Dir(distPath))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /backoffice prefix from path (since handler is registered at /backoffice/*)
		filePath := r.URL.Path
		if strings.HasPrefix(filePath, "/backoffice") {
			filePath = strings.TrimPrefix(filePath, "/backoffice")
		}
		if filePath == "" {
			filePath = "/"
		}

		// Clean the path
		filePath = filepath.Clean(filePath)
		// Remove leading slash so filepath.Join works correctly
		filePath = strings.TrimPrefix(filePath, "/")
		fullPath := filepath.Join(distPath, filePath)


		// Prevent directory traversal (normalize both paths for comparison)
		absDistPath, _ := filepath.Abs(distPath)
		absFullPath, _ := filepath.Abs(fullPath)
		if !strings.HasPrefix(absFullPath, absDistPath) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check if the file exists
		info, err := os.Stat(fullPath)

		if err == nil && !info.IsDir() {
			// Change r.URL.Path so fileServer knows what to serve
			// e.g., /backoffice/assets/file.js → /assets/file.js
			r.URL.Path = "/" + filePath
			fileServer.ServeHTTP(w, r)
			return
		}

		// If file doesn't exist and it's not an API request, serve index.html (SPA routing)
		if !isAPIPath(r.URL.Path) {
			// Serve index.html for SPA routing
			indexPath := filepath.Join(distPath, "index.html")
			file, err := os.Open(indexPath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer file.Close()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.Copy(w, file)
			return
		}

		// For API paths that don't exist, return 404
		http.NotFound(w, r)
	})
}

// isAPIPath checks if the request is for an API endpoint
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/v1") || strings.HasPrefix(path, "/webhooks")
}
