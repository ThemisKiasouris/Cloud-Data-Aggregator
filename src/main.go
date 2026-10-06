package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveRequestTimeout = 30 * time.Second

func main() {
	// command-line flags
	download := flag.Bool("d", false, "download the search results to a local file")
	keywordsFlag := flag.String("key", "", "comma-separated keywords to search for")
	folderFlag := flag.String("f", "", "Google Drive folder ID to scan")

	flag.Parse()

	if *download {
		fmt.Println("Download flag enabled. Search will still run in this version.")
	}

	targetKeywords, err := parseKeywords(*keywordsFlag)
	if err != nil {
		log.Fatalf("Invalid keywords: %v", err)
	}
	if len(targetKeywords) == 0 {
		log.Fatal("No keywords provided. Use -key \"keyword1,keyword2\".")
	}

	folderID := strings.TrimSpace(*folderFlag)
	if folderID == "" {
		log.Fatal("No folder ID provided. Use -f \"YOUR_FOLDER_ID\".")
	}

	fmt.Printf("\nInitializing scan for keywords: %v in folder: %s\n", targetKeywords, folderID)

	credsJSON, err := loadCredentialsJSON()
	if err != nil {
		log.Fatalf("Unable to load credentials: %v", err)
	}

	ctx := context.Background()

	config, err := google.ConfigFromJSON([]byte(credsJSON), drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	tok := getTokenFromWeb(config)
	client := config.Client(context.Background(), tok)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Drive client: %v", err)
	}

	scanFolder(ctx, srv, folderID, targetKeywords)

	if *download {
		outputFile, err := writeResults()
		if err != nil {
			log.Fatalf("Failed to write results: %v", err)
		} else {
			logf("Results written to: %s\n", outputFile)
		}
	}
}
