package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var outputMu sync.Mutex

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Interactive Keyword Prompt
	fmt.Print("Enter target keywords (comma-separated, e.g., Revenue, Q1, Deficit): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		log.Fatal("No keywords provided. Exiting.")
	}

	rawKeywords := strings.Split(input, ",")
	var targetKeywords []string
	for _, kw := range rawKeywords {
		targetKeywords = append(targetKeywords, strings.TrimSpace(kw))
	}

	// Interactive Folder ID Prompt
	fmt.Print("Enter the Drive Folder ID to scan: ")
	folderInput, _ := reader.ReadString('\n')
	folderID := strings.TrimSpace(folderInput)

	fmt.Printf("\nInitializing scan for keywords: %v in folder: %s\n", targetKeywords, folderID)

	// Drive API Initialization
	credsJSON, err := loadCredentialsJSON()
	if err != nil {
		log.Fatalf("Unable to load credentials: %v", err)
	}

	ctx := context.Background()

	// Parse the Client ID JSON for OAuth 2.0
	config, err := google.ConfigFromJSON([]byte(credsJSON), drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Get the token via browser prompt
	tok := getTokenFromWeb(config)
	client := config.Client(context.Background(), tok)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Drive client: %v", err)
	}

	// Fetch files from the user-defined folder, retrieving the mimeType
	query := fmt.Sprintf("'%s' in parents", folderID)
	r, err := srv.Files.List().
		Q(query).
		Fields("files(id, name, mimeType)").
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Do()

	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	if len(r.Files) == 0 {
		fmt.Println("No files found in the specified folder.")
		return
	}

	// Concurrent Processing of Files
	const maxConcurrentFiles = 4
	jobs := make(chan *drive.File, len(r.Files))
	var wg sync.WaitGroup

	for worker := 0; worker < maxConcurrentFiles; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				// Safely skip files that aren't spreadsheets (like PDFs or Word docs)
				if !strings.Contains(file.MimeType, "spreadsheet") && !strings.Contains(file.MimeType, "excel") {
					continue
				}
				logf("\n--- Scanning Spreadsheet: %s ---\n", file.Name)
				processExcelData(file, srv, targetKeywords)
			}
		}()
	}

	for _, file := range r.Files {
		jobs <- file
	}
	close(jobs)
	wg.Wait()
}

// Handles the OAuth 2.0 web prompt
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
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
	return tok
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

// Processes the Excel data for matches
func processExcelData(file *drive.File, srv *drive.Service, keywords []string) {
	var resp *http.Response
	var err error

	// Export native Google Sheets to Excel format, otherwise download directly
	if file.MimeType == "application/vnd.google-apps.spreadsheet" {
		resp, err = srv.Files.Export(file.Id, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").Download()
	} else {
		resp, err = srv.Files.Get(file.Id).Download()
	}

	if err != nil {
		logf("Skipping file %s: %v\n", file.Name, err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logf("Failed to read download for file %s: %v\n", file.Name, err)
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		logf("Failed to open file %s: %v\n", file.Name, err)
		return
	}
	defer f.Close()

	var allRows [][]string

	// Dynamically read every sheet
	for _, sheetName := range f.GetSheetList() {
		sheetRows, err := f.GetRows(sheetName)
		if err != nil {
			logf("Failed to read rows from sheet %s in file %s: %v\n", sheetName, file.Name, err)
			continue
		}
		allRows = append(allRows, sheetRows...)
	}

	matchCount := 0

	// Scan and aggregate
	for rowIndex, row := range allRows {
		for _, cellValue := range row {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(cellValue), strings.ToLower(keyword)) {
					logf("[Match] File: %s | Row %d: Triggered by '%s' (Cell Value: %s)\n", file.Name, rowIndex+1, keyword, cellValue)
					matchCount++
				}
			}
		}
	}

	if matchCount == 0 {
		logf("No matches found in %s.\n", file.Name)
	} else {
		logf("Total matches found in %s: %d\n", file.Name, matchCount)
	}
}

// Thread-safe logging function
func logf(format string, args ...any) {
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Printf(format, args...)
}
