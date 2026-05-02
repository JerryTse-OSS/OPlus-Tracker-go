// Designed by Jerry Tse
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type HTTPReaderAt struct {
	URL    string
	Client *http.Client
	Size   int64
}

func (r *HTTPReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= r.Size {
		return 0, io.EOF
	}
	end := off + int64(len(p)) - 1
	if end >= r.Size {
		end = r.Size - 1
	}

	req, err := http.NewRequest("GET", r.URL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "python-requests/2.31.0")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	resp, err := r.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadFull(resp.Body, p)
}

type OpexConfig struct {
	BusinessCode string `json:"businessCode"`
	OvlList      []struct {
		OvlMountPath string `json:"ovlMountPath"`
	} `json:"ovlList"`
}

func analyzeOpexFromURL(url string) {
	client := &http.Client{}

	// Step 1: Get Content-Length
	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	headReq.Header.Set("User-Agent", "python-requests/2.31.0")
	headResp, err := client.Do(headReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK && headResp.StatusCode != http.StatusPartialContent {
		fmt.Fprintf(os.Stderr, "Error: HTTP %d\n", headResp.StatusCode)
		os.Exit(1)
	}
	if headResp.Header.Get("Accept-Ranges") != "bytes" && headResp.ContentLength <= 0 {
		fmt.Fprintf(os.Stderr, "Hint: The remote server may not support Range requests, partial download unavailable.\n")
		os.Exit(1)
	}

	size := headResp.ContentLength
	readerAt := &HTTPReaderAt{
		URL:    url,
		Client: client,
		Size:   size,
	}

	// Step 2: Open ZIP reader
	zipReader, err := zip.NewReader(readerAt, size)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 416") || strings.Contains(strings.ToLower(err.Error()), "range") {
			fmt.Fprintf(os.Stderr, "Error: %v\nHint: The remote server may not support Range requests, partial download unavailable.\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error opening ZIP: %v\n", err)
		}
		os.Exit(1)
	}

	// Step 3: Find opex.cfg
	var cfgFile *zip.File
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "opex.cfg") {
			cfgFile = file
			break
		}
	}

	if cfgFile == nil {
		fmt.Fprintf(os.Stderr, "Error: opex.cfg not found in the ZIP file\n")
		os.Exit(1)
	}

	// Step 4: Read opex.cfg
	rc, err := cfgFile.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening opex.cfg: %v\n", err)
		os.Exit(1)
	}
	defer rc.Close()

	cfgData, err := io.ReadAll(rc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading opex.cfg: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Parse JSON
	var cfg OpexConfig
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: opex.cfg is not valid JSON\n%v\n", err)
		os.Exit(1)
	}

	// Extract info
	businessCode := cfg.BusinessCode
	var ovlPaths []string
	for _, item := range cfg.OvlList {
		if item.OvlMountPath != "" {
			ovlPaths = append(ovlPaths, item.OvlMountPath)
		}
	}

	if businessCode == "" && len(ovlPaths) == 0 {
		fmt.Println("Warning: businessCode or ovlMountPath fields not found")
	}

	if businessCode != "" && len(ovlPaths) > 0 {
		var pathStr string
		if len(ovlPaths) == 1 {
			pathStr = fmt.Sprintf(`"%s"`, ovlPaths[0])
		} else {
			var quoted []string
			for _, p := range ovlPaths {
				quoted = append(quoted, fmt.Sprintf(`"%s"`, p))
			}
			pathStr = strings.Join(quoted[:len(quoted)-1], ", ") + " and " + quoted[len(quoted)-1]
		}
		fmt.Printf("\n\"%s\" is used to fix issues with %s\n", businessCode, pathStr)
	} else {
		fmt.Println("Unable to generate complete analysis result")
	}

	fmt.Println("\nDetails:")
	fmt.Printf("Opex Code: %s\n", businessCode)
	fmt.Printf("ovlList count: %d\n", len(ovlPaths))
	if len(ovlPaths) > 0 {
		fmt.Println("ovlMountPath list:")
		for _, p := range ovlPaths {
			fmt.Printf("  - %s\n", p)
		}
	}
}

func main() {
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: opex_analyzer <URL>\n")
		fmt.Fprintf(os.Stderr, "Example:\n  opex_analyzer https://example.com/opex.zip\n")
		os.Exit(1)
	}

	url := args[0]
	if url == "" {
		fmt.Fprintf(os.Stderr, "Error: link cannot be empty\n")
		os.Exit(1)
	}

	analyzeOpexFromURL(url)
	fmt.Println("\nAnalysis completed!")
}
