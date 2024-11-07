package main

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Map from the application to the cron schedule it should run on
type Configuration struct {
	shouldRunApps        []string
	availableFiles       []string
	availablePythonFiles []string
}

func downloadConfig() Configuration {
	err := os.RemoveAll("tmp/")
	if err != nil {
		fmt.Println("Error removing /tmp/:", err)
		panic(err)
	}

	err = os.Mkdir("tmp/", os.FileMode(0755))
	if err != nil {
		fmt.Println("Error creating /tmp/:", err)
		panic(err)
	}

	appList := "https://raw.githubusercontent.com/dreammify/rapidpy/refs/heads/main/applications/index.txt"
	resp, err := http.Get(appList)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
		panic(err)
	}

	shouldRunApps := strings.Split(string(body), "\n")

	dirList := "https://api.github.com/repos/dreammify/rapidpy/git/trees/main?recursive=1"
	resp, err = http.Get(dirList)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
		panic(err)
	}

	paths := gjson.GetBytes(body, "tree.#.path")
	strPaths := make([]string, 0)
	for _, path := range paths.Array() {
		if strings.HasPrefix(path.String(), "applications/") {
			strPaths = append(strPaths, strings.TrimPrefix(path.String(), "applications/"))
		}
	}

	availablePythonFiles := make([]string, 0)
	for _, file := range strPaths {
		if strings.HasSuffix(file, ".py") {
			availablePythonFiles = append(availablePythonFiles, file)
		}
	}

	return Configuration{
		shouldRunApps:        shouldRunApps,
		availableFiles:       strPaths,
		availablePythonFiles: availablePythonFiles,
	}
}

func downloadApplicationFiles(cfg Configuration, app string) {
	fmt.Println("Downloading files for application:", app)

	applicationFiles := make([]string, 0)

	for _, file := range cfg.availableFiles {
		if strings.HasPrefix(file, app) {
			applicationFiles = append(applicationFiles, file)
		}
	}

	for _, file := range applicationFiles {
		fmt.Println("Downloading file:", file)
		resp, err := http.Get("https://raw.githubusercontent.com/dreammify/rapidpy/main/applications/" + file)
		if err != nil {
			fmt.Println("Error fetching the URL:", err)
			panic(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading the response body:", err)
			panic(err)
		}

		err = os.WriteFile("tmp/"+file, body, os.FileMode(0777))
		if err != nil {
			fmt.Println("Error writing the file:", err)
			panic(err)
		}
	}
}

func pythonCommand(app string) exec.Cmd {
	if os.Getenv("RAPIDPY_ENV") == "PROD" {
		return *exec.Command("python3", fmt.Sprintf("./tmp/%s.py", app))
	} else {
		return *exec.Command(
			"bash",
			"-c", fmt.Sprintf("source ./venv/bin/activate && python3 ./tmp/%s", app))
	}
}

func runCommand(cmd exec.Cmd) error {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}

	return nil
}

var appChannels = make(map[string]chan string)

func main() {
	for {
		cfg := downloadConfig()
		fmt.Println(cfg)

		for _, app := range cfg.availablePythonFiles {
			if _, ok := appChannels[app]; ok {
				continue
			} else {
				fmt.Println("Starting application:", app)
				downloadApplicationFiles(cfg, app)

				go func() {
					appChannels[app] = make(chan string)
					_, ok := appChannels[app]
					for ok {
						cmd := pythonCommand(app)
						runCommand(cmd)
					}
				}()
			}
		}

		time.Sleep(10 * time.Second)
	}

}
