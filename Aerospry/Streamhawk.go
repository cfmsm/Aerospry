package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

const ChunkSizeMB = 5
const MaxConcurrency = 32

var buf = make([]byte, 64*1024) // bigger buffer for faster writes

func main() {
	reader := bufio.NewReader(os.Stdin)
	rawURL, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read input:", err)
		return
	}
	rawURL = strings.TrimSpace(rawURL)

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		return
	}

	fileName := path.Base(parsedURL.Path)
	if fileName == "" || fileName == "/" {
		fileName = "downloaded_file"
	}

	// Get file size
	resp, err := http.Head(rawURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch headers:", err, resp.Status)
		return
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		fmt.Println("Cannot determine file size:", err)
		return
	}
	fmt.Printf("File size: %d bytes\n", size)

	// Preallocate final file
	finalFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating final file:", err)
		return
	}
	defer finalFile.Close()
	if err := finalFile.Truncate(int64(size)); err != nil {
		fmt.Println("Error preallocating file:", err)
		return
	}

	// Calculate chunks
	chunkSize := ChunkSizeMB * 1024 * 1024
	numChunks := (size + chunkSize - 1) / chunkSize
	fmt.Printf("Downloading in %d chunks of %dMB each...\n", numChunks, ChunkSizeMB)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1024,
			MaxIdleConnsPerHost: 1024, // enable many parallel connections
		},
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxConcurrency)

	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if end >= size {
			end = size - 1
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := downloadDirect(client, rawURL, finalFile, start, end); err != nil {
				fmt.Printf("Error downloading chunk %d-%d: %v\n", start, end, err)
			}
		}(start, end)
	}

	wg.Wait()
	fmt.Printf("Download complete: '%s'\n", fileName)
}

func downloadDirect(client *http.Client, url string, file *os.File, start, end int) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	_, err = io.CopyBuffer(NewWriterAt(file, int64(start)), resp.Body, buf)
	return err
}

// WriterAt wrapper for io.Writer interface
type WriterAt struct {
	f   *os.File
	pos int64
}

func NewWriterAt(f *os.File, pos int64) *WriterAt {
	return &WriterAt{f: f, pos: pos}
}

func (w *WriterAt) Write(p []byte) (int, error) {
	n, err := w.f.WriteAt(p, w.pos)
	w.pos += int64(n)
	return n, err
}
