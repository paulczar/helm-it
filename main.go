// main.go
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/slok/go-helm-template/helm"
)

// RequestPayload defines the structure of the incoming JSON request.
// It expects a URL to a .tgz Helm chart and an optional map of values to override.
type RequestPayload struct {
	ChartURL string                 `json:"chartUrl"`
	Values   map[string]interface{} `json:"values,omitempty"`
}

// ResponsePayload defines the structure of the JSON response.
type ResponsePayload struct {
	Templates   string `json:"templates"`
	Values      string `json:"values,omitempty"`
	ValuesExist bool   `json:"valuesExist"`
}

// downloadChart downloads a Helm chart from a given URL to a temporary directory.
// It returns the file path to the downloaded chart and an error if one occurs.
func downloadChart(url string) (string, error) {
	// Create a temporary file to store the downloaded chart.
	tempDir, err := os.MkdirTemp("", "helm-template-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	tempFilePath := filepath.Join(tempDir, filepath.Base(url))
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Perform the HTTP GET request to download the chart.
	log.Printf("Downloading chart from %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download chart: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download chart, status code: %d", resp.StatusCode)
	}

	// Copy the downloaded content to the temporary file.
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy content to file: %w", err)
	}

	log.Printf("Successfully downloaded chart to %s\n", tempFilePath)
	return tempFilePath, nil
}

// extractTarball extracts a .tgz file into a new temporary directory.
func extractTarball(tarballPath string) (string, error) {
	// Open the gzipped tarball.
	file, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Create a new temporary directory for extraction.
	tempExtractDir, err := os.MkdirTemp("", "helm-extract-")
	if err != nil {
		return "", fmt.Errorf("failed to create extraction directory: %w", err)
	}

	// Iterate through the files in the tarball and extract them.
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(tempExtractDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists.
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return "", fmt.Errorf("failed to copy file content: %w", err)
			}
			outFile.Close()
		}
	}

	return tempExtractDir, nil
}

// templateHandler is the HTTP handler for the /template endpoint.
func templateHandler(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	var err error

	// Handle different HTTP methods.
	switch r.Method {
	case http.MethodPost:
		// Decode the JSON request body.
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}
	case http.MethodGet:
		// Get chartUrl and optional values from query parameters.
		chartURL := r.URL.Query().Get("chartUrl")
		if chartURL == "" {
			http.Error(w, "Missing 'chartUrl' query parameter", http.StatusBadRequest)
			return
		}
		payload.ChartURL = chartURL

		// Decode JSON-encoded values from the query parameter if present.
		valuesParam := r.URL.Query().Get("values")
		if valuesParam != "" {
			err = json.Unmarshal([]byte(valuesParam), &payload.Values)
			if err != nil {
				http.Error(w, "Invalid 'values' query parameter. Must be a JSON string.", http.StatusBadRequest)
				return
			}
		}

	default:
		http.Error(w, "Only POST and GET methods are supported", http.StatusMethodNotAllowed)
		return
	}

	// Validate the chart URL.
	if !strings.HasSuffix(payload.ChartURL, ".tgz") {
		http.Error(w, "Invalid or missing 'chartUrl'. Must be a .tgz URL.", http.StatusBadRequest)
		return
	}

	// Check for the 'raw' query parameter to determine the output format.
	rawQuery := r.URL.Query().Get("raw")
	renderJSON := true
	if rawQuery == "true" {
		renderJSON = false
	}

	templateChartAndRender(w, r, payload, renderJSON)
}

// templateChartAndRender handles the core logic of templating a Helm chart.
func templateChartAndRender(w http.ResponseWriter, r *http.Request, payload RequestPayload, renderJSON bool) {
	// Download the Helm chart.
	tempFilePath, err := downloadChart(payload.ChartURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error downloading chart: %s", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(filepath.Dir(tempFilePath))

	// Extract the downloaded tarball.
	extractDir, err := extractTarball(tempFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error extracting chart: %s", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(extractDir)

	// Find the actual chart directory inside the extracted folder.
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading extracted directory: %s", err), http.StatusInternalServerError)
		return
	}

	var chartRootDir string
	for _, entry := range entries {
		if entry.IsDir() {
			chartRootDir = filepath.Join(extractDir, entry.Name())
			break
		}
	}

	if chartRootDir == "" {
		http.Error(w, "Could not find a valid chart directory in the tarball", http.StatusInternalServerError)
		return
	}

	// Read the values.yaml file if it exists.
	valuesFilePath := filepath.Join(chartRootDir, "values.yaml")
	var valuesContent string
	valuesExist := true
	valuesBytes, err := os.ReadFile(valuesFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Error reading values.yaml: %s", err), http.StatusInternalServerError)
			return
		}
		valuesExist = false
	} else {
		valuesContent = string(valuesBytes)
	}

	// Load the chart from the extracted directory.
	ctx := context.Background()
	chartFS := os.DirFS(chartRootDir)
	loadedChart, err := helm.LoadChart(ctx, chartFS)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading chart: %s", err), http.StatusInternalServerError)
		return
	}

	// Define the configuration for templating.
	config := helm.TemplateConfig{
		Chart:       loadedChart,
		ReleaseName: "my-release",
		Namespace:   "default",
		Values:      payload.Values,
	}

	// Execute the Helm template rendering.
	result, err := helm.Template(ctx, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error templating chart: %s", err), http.StatusInternalServerError)
		return
	}

	if renderJSON {
		response := ResponsePayload{
			Templates:   result,
			Values:      valuesContent,
			ValuesExist: valuesExist,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	chartURL := r.URL.Query().Get("c")
	if chartURL != "" {
		// If 'c' param exists, treat it as a raw template request.
		payload := RequestPayload{ChartURL: chartURL}
		templateChartAndRender(w, r, payload, false) // false for raw output
	} else {
		// Otherwise, serve the static files.
		fileServer().ServeHTTP(w, r)
	}
}

func main() {
	// Define the HTTP handler.
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/template", templateHandler)

	// Start the server on port 8080.
	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
