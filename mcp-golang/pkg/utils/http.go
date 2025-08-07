package utils

import (
	"io"
	"net/http"
)

func HttpGet(url string, header map[string]string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
