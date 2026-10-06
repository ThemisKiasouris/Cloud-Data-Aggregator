package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/xuri/excelize/v2"
	"google.golang.org/api/drive/v3"
)

// Results struct to hold the search results
type MatchResults struct {
	FileName string
	RowNum   int
	Keyword  string
	CellVal  string
}

var (
	resultsMu sync.Mutex
	results   []MatchResults
)

func addResult(r MatchResults) {
	resultsMu.Lock()
	defer resultsMu.Unlock()
	results = append(results, r)
}

// Processes the Excel data for matches
func processExcelData(ctx context.Context, file *drive.File, srv *drive.Service, keywords []Keyword) {
	reqCtx, cancel := context.WithTimeout(ctx, driveRequestTimeout)
	defer cancel()

	var resp *http.Response
	var err error

	// Export native Google Sheets to Excel format, otherwise download directly
	if file.MimeType == "application/vnd.google-apps.spreadsheet" {
		resp, err = srv.Files.Export(file.Id, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").Context(reqCtx).Download()
	} else {
		resp, err = srv.Files.Get(file.Id).Context(reqCtx).Download()
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
				if keyword.Re.MatchString(cellValue) {
					logf("[Match] File: %s | Row %d: Triggered by '%s' (Cell Value: %s)\n", file.Name, rowIndex+1, keyword.Raw, cellValue)
					addResult(MatchResults{
						FileName: file.Name,
						RowNum:   rowIndex + 1,
						Keyword:  keyword.Raw,
						CellVal:  cellValue,
					})
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
