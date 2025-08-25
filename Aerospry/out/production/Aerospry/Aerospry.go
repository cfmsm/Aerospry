package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const ChunkSizeMB = 7
const MaxConcurrency = 256

var buf = make([]byte, 64*1024) // bigger buffer for faster writes

var (
	home, _ = os.UserHomeDir()
	user    = os.Getenv("USER")
	osName  = runtime.GOOS
)

func main() {
	fmt.Println("\n\n")
	if osName == "darwin" {
		fmt.Println("\uF8FF")
	} else if osName == "windows" {
		fmt.Println("\u229E")
	} else if osName == "linux" {
		fmt.Println("\U0001F427")
	}
	fmt.Println("Hello,", user)
	fmt.Println("\nAerospry successfully booted")
	fmt.Println("Its a combination of Polymall and Streamhawk.")
	fmt.Println("Note: If it shows where the file is saved and it's an archive, you should extract it")
	fmt.Println("Type 'streamhawk help' if you don't know what to do. Its simple :)")
	fmt.Println("Type 'polymall help' if you don't know what to do. Its simple :)")

	command()
}

// ========================
// Polymall Commands
// ========================
func command() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(in, "polymall install "):
			name := strings.TrimSpace(strings.TrimPrefix(in, "polymall install "))
			url := "https://raw.githubusercontent.com/cfmsm/Polymall/main/downloads/recipes/" + name + "/temp.cfg"
			install(url)

		case strings.HasPrefix(in, "streamhawk "):
			link := strings.TrimSpace(strings.TrimPrefix(in, "streamhawk "))
			if link == "" {
				fmt.Println("Please provide a URL after 'streamhawk'")
				continue
			}
			fn := path.Base(link)
			fmt.Println("Downloading via Streamhawk:", link)
			if err := downloadFile(link, fn); err != nil {
				fmt.Println("Download failed:", err)
			}

		case strings.HasPrefix(in, "polymall help"):
			fmt.Println("Use 'polymall install' to install a package")
			fmt.Println("Type the name of the package at the end of 'polymall install' command")
			fmt.Println("Example: polymall install quellwrap")
			fmt.Println("Use 'streamhawk <url>' to download a file directly")
			fmt.Println("Use 'polymall list' to search for packages")
			fmt.Println("Use 'polymall issue' to get solutions to common problems")
			fmt.Println("Use 'polymall exit' to exit the software")
		case in == "polymall exit":
			return
		case strings.HasPrefix(in, "streamhawk help"):
			fmt.Println("Use 'streamhawk <url>' to download a file directly")
			fmt.Println("Example: streamhawk https://example.com/file.zip")
			return
		default:
			fmt.Println("Unknown command. Type 'polymall help' for instructions.")
		}
	}
}

func install(cfgUrl string) {
	tmp := "temp"
	os.MkdirAll(tmp, os.ModePerm)

	fileName := filepath.Base(cfgUrl)
	cfgPath := filepath.Join(tmp, fileName)
	fmt.Println("Downloading config:", cfgUrl)
	if err := downloadFile(cfgUrl, cfgPath); err != nil {
		fmt.Println("Failed to download config:", err)
		return
	}

	cfgMap, err := readCfg(cfgPath)
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	// Ensure all URLs have a protocol
	for k, list := range cfgMap {
		for i, v := range list {
			if needsProto(v) {
				list[i] = "https://" + v
			}
		}
		cfgMap[k] = list
	}

	key := osKey()
	dirKey := "downloads"
	if dirs, ok := cfgMap["dir"+capitalize(key)]; ok && len(dirs) > 0 {
		dirKey = dirs[0]
	} else if dirs, ok := cfgMap["dir"]; ok && len(dirs) > 0 {
		dirKey = dirs[0]
	}

	saveDir := usrDir(dirKey)
	os.MkdirAll(saveDir, os.ModePerm)

	links := cfgMap[key]
	if len(links) == 0 {
		links = cfgMap["all"]
		if len(links) == 0 {
			fmt.Println("No '" + key + "' or 'all' entry found in config")
			return
		}
	}

	for _, url := range links {
		fn := filepath.Base(url)
		out := filepath.Join(saveDir, fn)
		fmt.Println("Downloading from:", url)
		if err := downloadFile(url, out); err != nil {
			fmt.Println("Failed to download:", err)
		}
	}

	os.Remove(cfgPath)
	os.Remove(tmp)

	if _, ok := cfgMap["open"]; ok {
		openDir(saveDir)
	}

	fmt.Println("\n\033[32m==> DOWNLOAD SUCCESSFUL!\033[0m")
}

func readCfg(path string) (map[string][]string, error) {
	cfg := make(map[string][]string)
	file, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, "\r", "")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx >= 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			cfg[key] = append(cfg[key], val)
		} else {
			key := line
			if _, ok := cfg[key]; !ok {
				cfg[key] = []string{}
			}
		}
	}
	return cfg, scanner.Err()
}

func usrDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(home, strings.ReplaceAll(dir, "+", ""))
}

func osKey() string {
	switch {
	case strings.Contains(osName, "windows"):
		return "win"
	case strings.Contains(osName, "darwin"):
		return "mac"
	default:
		return "nux"
	}
}

func openDir(dir string) {
	var cmd *exec.Cmd
	switch osKey() {
	case "win":
		cmd = exec.Command("explorer", dir)
	case "mac":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	cmd.Start()
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func needsProto(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	return strings.Contains(t, ".") && !strings.HasPrefix(t, "http://") && !strings.HasPrefix(t, "https://")
}

// ========================
// Multi-Chunk Downloader
// ========================
func downloadFile(rawURL, fileName string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	if fileName == "" {
		fileName = path.Base(parsedURL.Path)
		if fileName == "" || fileName == "/" {
			fileName = "downloaded_file"
		}
	}

	// Get file size
	resp, err := http.Head(rawURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch headers: %v %s", err, resp.Status)
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return fmt.Errorf("cannot determine file size: %v", err)
	}
	fmt.Printf("File size: %d bytes\n", size)

	// Preallocate final file
	finalFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating final file: %v", err)
	}
	defer finalFile.Close()
	if err := finalFile.Truncate(int64(size)); err != nil {
		return fmt.Errorf("error preallocating file: %v", err)
	}

	// Calculate chunks
	chunkSize := ChunkSizeMB * 1024 * 1024
	numChunks := (size + chunkSize - 1) / chunkSize
	fmt.Printf("Downloading in %d chunks of %dMB each...\n", numChunks, ChunkSizeMB)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1024,
			MaxIdleConnsPerHost: 1024,
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

			if err := downloadChunk(client, rawURL, finalFile, start, end); err != nil {
				fmt.Printf("Error downloading chunk %d-%d: %v\n", start, end, err)
			}
		}(start, end)
	}

	wg.Wait()
	fmt.Printf("Download complete: '%s'\n", fileName)
	return nil
}

func downloadChunk(client *http.Client, url string, file *os.File, start, end int) error {
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
