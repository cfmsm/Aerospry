package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

var knownHashes = map[string]string{
	"ffmpeg": "26da33fccea746d177d5b69f760922f7c23bc33f87cd06d10d581214a1d28d85",
}

// getFileHash computes SHA256 hash of a file
func getFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	// ask for path
	fmt.Print("Enter the file path: ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)

	// ask for package name
	fmt.Print("Enter the package name: ")
	packageName, _ := reader.ReadString('\n')
	packageName = strings.TrimSpace(packageName)

	// look up known hash
	expectedHash, exists := knownHashes[packageName]
	if !exists {
		fmt.Printf("No known hash for package: %s\n", packageName)
		os.Exit(1)
	}

	// compute file hash
	computedHash, err := getFileHash(path)
	if err != nil {
		fmt.Println("Error computing hash:", err)
		os.Exit(1)
	}

	// show results
	fmt.Println("Expected Hash:", expectedHash)
	fmt.Println("Computed Hash:", computedHash)

	if computedHash == expectedHash {
		fmt.Println("✅ Verification passed for", packageName)
	} else {
		fmt.Println("❌ Verification failed for", packageName)
	}
}
