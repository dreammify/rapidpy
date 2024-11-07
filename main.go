package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Map from the application to the cron schedule it should run on
type Configuration struct {
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

	dirList := "https://api.github.com/repos/dreammify/rapidpy/git/trees/main?recursive=1"
	req, err := http.NewRequest("GET", dirList, nil)
	if err != nil {
		fmt.Println("Error creating the request:", err)
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GITHUB_PAT")))
	resp, err := http.DefaultClient.Do(req)
	fmt.Println(resp.Header.Get("X-RateLimit-Remaining"))
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

func runCommand(cmd exec.Cmd) (*os.Process, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return nil, err
	}

	return cmd.Process, nil
}

var appShouldRun = make(map[string]bool)

func readAppRunConfig(app string) int {
	file, err := os.Open("./tmp/" + app)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	cfg := scanner.Text()
	cfi, err := strconv.Atoi(strings.TrimPrefix(cfg, "# "))
	if err != nil {
		panic(err)
	}
	return cfi
}

func appEventLoop(app string, appShouldRun map[string]bool) {
	runConfig := readAppRunConfig(app)
	appShouldRun[app] = true
	if runConfig == 0 {
		fmt.Println("Running continuous application: ", app)
		cmd := pythonCommand(app)
		pid, err := runCommand(cmd)
		if err != nil {
			fmt.Println("Error starting the application:", err)
			delete(appShouldRun, app)
		}

		for appShouldRun[app] {
			time.Sleep(1 * time.Second)
		}

		err = pid.Kill()
		if err != nil {
			fmt.Println("Error stopping the application:", err)
		}
	} else {
		fmt.Println("Running recurrent application: ", app, "every", runConfig, "seconds")
		for appShouldRun[app] {
			cmd := pythonCommand(app)
			_, err := runCommand(cmd)
			if err != nil {
				fmt.Println("Error starting the application:", err)
				delete(appShouldRun, app)
			}

			time.Sleep(time.Duration(runConfig) * time.Second)
		}
	}

}

func main() {
	for {
		cfg := downloadConfig()
		fmt.Println(cfg)

		for _, app := range cfg.availablePythonFiles {
			if _, ok := appShouldRun[app]; ok {
				continue
			} else {
				fmt.Println("Starting application:", app)
				downloadApplicationFiles(cfg, app)
				go appEventLoop(app, appShouldRun)
			}
		}

		for app := range appShouldRun {
			if !slices.Contains(cfg.availablePythonFiles, app) {
				fmt.Println("Stopping application:", app)
				appShouldRun[app] = false
				delete(appShouldRun, app)
			}
		}

		time.Sleep(10 * time.Second)
	}
}
