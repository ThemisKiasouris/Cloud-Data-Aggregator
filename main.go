package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Interactive Keyword Prompt
	fmt.Print("Enter target keywords (comma-separated, e.g., Revenue, Q1, Deficit): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	// clean and parse the input into a slice
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
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// Fetch files from the user-defined folder
	query := fmt.Sprintf("'%s' in parents and mimeType='application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'", folderID)
	r, err := srv.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	// Process each file
	for _, i := range r.Files {
		fmt.Printf("\n--- Scanning Spreadsheet: %s ---\n", i.Name)
		processExcelData(i.Id, srv, targetKeywords)
	}
}

func processExcelData(fileID string, srv *drive.Service, keywords []string) {

	f, err := excelize.OpenFile("temp_downloaded_file.xlsx")
	if err != nil {
		log.Printf("Failed to open file %s: %v", fileID, err)
		return
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		log.Printf("Failed to read rows: %v", err)
		return
	}

	matchCount := 0

	// Scan and aggregate based on dynamic keywords
	for rowIndex, row := range rows {
		for _, cellValue := range row {
			for _, keyword := range keywords {
				// Using case-insensitive matching for better general utility
				if strings.Contains(strings.ToLower(cellValue), strings.ToLower(keyword)) {
					fmt.Printf("[Match] Row %d: Triggered by '%s' (Cell Value: %s)\n", rowIndex+1, keyword, cellValue)
					matchCount++
				}
			}
		}
	}

	if matchCount == 0 {
		fmt.Println("No matches found in this file.")
	} else {
		fmt.Printf("Total matches found: %d\n", matchCount)
	}
}
