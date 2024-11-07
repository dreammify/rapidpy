package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://raw.githubusercontent.com/dreammify/rapidpy/refs/heads/main/applications/index.txt"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
		return
	}

	fmt.Println(string(body))
}
