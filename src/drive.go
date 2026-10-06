package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"google.golang.org/api/drive/v3"
)

func listAllFiles(ctx context.Context, srv *drive.Service, folderID string) ([]*drive.File, error) {
	var allFiles []*drive.File
	query := fmt.Sprintf("'%s' in parents", folderID)
	pageToken := ""

	for {
		listCtx, listCancel := context.WithTimeout(ctx, driveRequestTimeout)

		listCall := srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType)").
			IncludeItemsFromAllDrives(true).
			SupportsAllDrives(true).
			PageSize(100).
			Context(listCtx)

		if pageToken != "" {
			listCall = listCall.PageToken(pageToken)
		}

		r, err := listCall.Do()
		listCancel()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve files: %w", err)
		}

		allFiles = append(allFiles, r.Files...)

		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}

	return allFiles, nil
}

func scanFolder(ctx context.Context, srv *drive.Service, folderID string, targetKeywords []Keyword) {
	allFiles, err := listAllFiles(ctx, srv, folderID)
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	if len(allFiles) == 0 {
		fmt.Println("No files found in the specified folder.")
		return
	}

	const maxConcurrentFiles = 4
	jobs := make(chan *drive.File, len(allFiles))
	var wg sync.WaitGroup

	for worker := 0; worker < maxConcurrentFiles; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				if !strings.Contains(file.MimeType, "spreadsheet") && !strings.Contains(file.MimeType, "excel") {
					continue
				}
				logf("\n--- Scanning Spreadsheet: %s ---\n", file.Name)
				processExcelData(ctx, file, srv, targetKeywords)
			}
		}()
	}

	for _, file := range allFiles {
		jobs <- file
	}
	close(jobs)
	wg.Wait()
}
