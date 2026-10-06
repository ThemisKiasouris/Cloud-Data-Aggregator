package main

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// writes all collected matches to a new .xlsx file
// and returns the path it wrote to.
func writeResults() (string, error) {
	resultsMu.Lock()
	defer resultsMu.Unlock()

	if len(results) == 0 {
		logf("No matches found, skipping writing results file.\n")
	}

	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Results"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"File Name", "Row Number", "Keyword", "Cell Value"}

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, r := range results {
		row := i + 2 // Start writing from the second row
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.FileName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.RowNum)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Keyword)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CellVal)

	}

	// Save the Excel file with a timestamped filename
	filename := fmt.Sprintf("scan-results-%s.xlsx", time.Now().Format("20060102-150405"))
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("failed to save results file: %w", err)
	}

	return filename, nil
}
