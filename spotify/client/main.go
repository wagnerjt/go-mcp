package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func getEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}

func main() {
	var token string = getEnv("SPOTIFY_TOKEN")
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		fmt.Errorf("error creating request: %w", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Errorf("error making request: %w", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("unexpected status code: got %v want %v", resp.StatusCode, http.StatusOK)
		return
	}
	fmt.Println("Successfully fetched user data from Spotify API")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Errorf("error reading response body: %w", err)
		return
	}
	fmt.Println("Response body:", string(body))
}
