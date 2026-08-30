package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

// Handles token retrieval: cache first, browser prompt as fallback
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	if tok, err := LoadCachedToken(); err == nil {
		if tok.Valid() {
			return tok
		}
		// Expired but has a refresh token — refresh silently
		if tok.RefreshToken != "" {
			src := config.TokenSource(context.Background(), tok)
			newTok, err := src.Token()
			if err == nil {
				SaveToken(newTok)
				return newTok
			}
			log.Printf("Warning: token refresh failed, falling back to browser auth: %v", err)
		}
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)
	fmt.Print("Enter the code here: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	SaveToken(tok)
	return tok
}

// Cache for Authorization and Token
func cacheToken() string {
	return filepath.Join(os.TempDir(), "token.json")
}

// Load cached token from file
func LoadCachedToken() (*oauth2.Token, error) {
	f, err := os.Open(cacheToken())
	if err != nil {
		return nil, err
	}

	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// save token to file
func SaveToken(token *oauth2.Token) error {
	f, err := os.OpenFile(cacheToken(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Warning: unable to cache OAuth token: %v", err)
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

// Loads credentials
func loadCredentialsJSON() (string, error) {
	if creds := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_CREDENTIALS")); creds != "" {
		return creds, nil
	}

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return "", fmt.Errorf("failed to load .env: %w", err)
		}
	}

	if creds := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_CREDENTIALS")); creds != "" {
		return creds, nil
	}

	if _, err := os.Stat("credentials.json"); err == nil {
		data, err := os.ReadFile("credentials.json")
		if err != nil {
			return "", fmt.Errorf("failed to read credentials.json: %w", err)
		}
		return string(data), nil
	}

	return "", fmt.Errorf("no credentials found")
}
