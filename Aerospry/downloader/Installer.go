package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func downloadFile(url string, filename string) error {
	// Send GET request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Construct full file path in home directory
	filePath := filepath.Join(homeDir, filename)

	// Create file to save
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Make file executable on Unix-like systems
	if runtime.GOOS != "windows" {
		err = os.Chmod(filePath, 0755)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Downloaded %s to %s\n", filename, filePath)
	return nil
}

func main() {
	urls := []string{
		"https://github.com/cfmsm/Polymall/raw/refs/heads/main/downloads/Aerospry",
		"https://github.com/cfmsm/Polymall/raw/refs/heads/main/downloads/Aerospry%20Verifier",
	}
	filenames := []string{
		"Aerospry",
		"Aerospry Verifier",
	}

	for i, url := range urls {
		err := downloadFile(url, filenames[i])
		if err != nil {
			fmt.Printf("Error downloading %s: %v\n", url, err)
		}
	}
}
