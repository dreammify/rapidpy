package main

import (
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"os"
	"strings"
)

// Map from the application to the cron schedule it should run on
var runningApplications = make(map[string]string)

type Configuration struct {
	shouldRunApps  []string
	availableFiles []string
}

func initialize() Configuration {
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

	return Configuration{
		shouldRunApps:  shouldRunApps,
		availableFiles: strPaths,
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

		err = os.WriteFile("tmp/"+file, body, os.FileMode(0644))
		if err != nil {
			fmt.Println("Error writing the file:", err)
			panic(err)
		}
	}

}

func main() {
	cfg := initialize()
	fmt.Println(cfg)
	for _, app := range cfg.shouldRunApps {
		downloadApplicationFiles(cfg, app)
	}
}
