package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	home, _ = os.UserHomeDir()
	user    = os.Getenv("USER")
	osName  = runtime.GOOS
)

func main() {
	fmt.Println("\n\n\nHello,", user)
	fmt.Println("\nPolymall successfully booted")
	fmt.Println("Note: If it shows where the file is saved and it's an archive, you should extract it")
	fmt.Println("Type 'polymall help' if you don't know what to do. Its simple :)")

	command()
}

func install(cfgUrl string) {
	tmp := "temp"
	os.MkdirAll(tmp, os.ModePerm)

	fileName := filepath.Base(cfgUrl)
	cfgPath := filepath.Join(tmp, fileName)
	download(cfgUrl, cfgPath)

	cfgMap, err := readCfg(cfgPath)
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	// Add https:// if needed
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
		download(url, out)
	}

	os.Remove(cfgPath)
	os.Remove(tmp)

	if _, ok := cfgMap["open"]; ok {
		openDir(saveDir)
	}

	fmt.Println("\n\033[32m==> DOWNLOAD SUCCESSFUL!\033[0m")
}

func needsProto(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	return strings.Contains(t, ".") && !strings.HasPrefix(t, "http://") && !strings.HasPrefix(t, "https://")
}

func download(url, outPath string) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Download failed:", err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Println("Cannot create file:", err)
		return
	}
	defer out.Close()

	io.Copy(out, resp.Body)
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

func command() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(in, "polymall install "):
			name := strings.TrimSpace(strings.TrimPrefix(in, "polymall install "))
			url := "https://raw.githubusercontent.com/cfmsm/Polymall/main/downloads/recipes/" + name + "/temp.cfg"
			install(url)
		case strings.HasPrefix(in, "polymall help"):
			fmt.Println("Use 'polymall install' to install a package")
			fmt.Println("Type the name of the package at the end of 'polymall install' command")
			fmt.Println("Example: polymall install quellwrap")
			fmt.Println("This will install QuellWrap")
			fmt.Println("Use 'polymall list' to search for packages")
			fmt.Println("Use 'polymall issue' to get solutions to common problems")
			fmt.Println("Use 'polymall exit' to exit the software")
		case in == "polymall exit":
			return
		default:
			fmt.Println("Unknown command. Type 'polymall help' for instructions.")
		}
	}
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
