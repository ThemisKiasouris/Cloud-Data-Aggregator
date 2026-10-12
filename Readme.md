# Cloud Data Aggregator

A command-line tool written in Go that scans every spreadsheet in a Google Drive folder for whole-word keyword matches, and can export the results to an Excel file.

It handles both native Google Sheets and uploaded `.xlsx` files, scans all sheets (tabs) in each workbook, and processes files concurrently.

## Features

- **Whole-word, case-insensitive matching**: searching for `Active` matches `active` and `ACTIVE.`, but not `Inactive` or `Activate`
- **Multiple keywords** in a single run
- **Google Sheets and Excel support**: native Sheets are exported to `.xlsx` on the fly
- **All tabs scanned** in every workbook
- **Concurrent scanning** with a bounded worker pool (4 files at a time)
- **Pagination**: folders with more than 100 files are fully scanned
- **Request timeouts**: each Drive request has a 30-second deadline, so a hung connection can't stall the run
- **OAuth token caching**: sign in once; later runs reuse (and silently refresh) the saved token
- **Excel export** of all matches with the `-d` flag
- **Read-only access**: requests only the `drive.readonly` scope and never modifies your files

## Requirements

- Go 1.18 or newer
- A Google account with access to the target folder
- A Google Cloud project with the Drive API enabled (setup below)

## Setup

### 1. Create Google OAuth credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/) and create (or select) a project.
2. Enable the **Google Drive API** under *APIs & Services → Library*.
3. Configure the **OAuth consent screen**. If the app is in testing mode, add your Google account as a test user.
4. Under *APIs & Services → Credentials*, choose *Create Credentials → OAuth client ID* and select **Desktop app**.
5. Download the JSON file.

### 2. Provide the credentials

The tool looks for credentials in this order:

1. The `GOOGLE_DRIVE_CREDENTIALS` environment variable (the full JSON contents)
2. A `.env` file containing `GOOGLE_DRIVE_CREDENTIALS=...`
3. A `credentials.json` file in the working directory

The simplest option is to save the downloaded file as `credentials.json` next to the source code. See `.env.example` for the `.env` format.

> **Never commit `credentials.json` or `.env`.** Make sure both are listed in `.gitignore`.

### 3. Install dependencies

```bash
go mod download
```

## Usage

```bash
go run . -key "keyword1,keyword2" -f "FOLDER_ID" [-d]
```

| Flag   | Required | Description                                                   |
| ------ | -------- | ------------------------------------------------------------- |
| `-key` | Yes      | Comma-separated keywords to search for                        |
| `-f`   | Yes      | Google Drive folder ID to scan                                |
| `-d`   | No       | Save all matches to a timestamped `.xlsx` file when finished  |

> Run with `go run .` (the whole package), not `go run main.go`. The code is split across several files, and naming a single file compiles only that one.

### Finding the folder ID

Open the folder in Google Drive. The ID is the last part of the URL:

```
https://drive.google.com/drive/folders/1JtpCNkrnf9lY4mrayFnaPJWHoNOiNTXr
                                       └────────── folder ID ──────────┘
```

### Examples

```bash
# Search for a single keyword
go run . -key "Active" -f "1JtpCNkrnf9lY4mrayFnaPJWHoNOiNTXr"

# Search for several keywords and export the results
go run . -key "Active,Pending,Overdue" -f "1JtpCNkrnf9lY4mrayFnaPJWHoNOiNTXr" -d
```

### Building a binary

```bash
go build -o data-aggregator.exe .
./data-aggregator.exe -key "Active" -f "YOUR_FOLDER_ID"
```

### First run: authorization

On the first run the tool prints an authorization URL:

1. Open the link in your browser and approve access.
2. Your browser will be redirected to `http://localhost/?code=...`. The page may fail to load, which is expected.
3. Copy the value of the `code` parameter from the address bar and paste it into the terminal.

The token is cached in your OS temp directory as `token.json` and reused on later runs. If it expires, it is refreshed automatically. Delete the file to force a fresh sign-in.

## Output

### Console

Each match is printed as it is found, followed by a per-file summary:

```
--- Scanning Spreadsheet: Employees Q3 ---
[Match] File: Employees Q3 | Row 14: Triggered by 'Active' (Cell Value: Active)
Total matches found in Employees Q3: 1
```

### Excel export (`-d`)

Creates `scan-results-YYYYMMDD-HHMMSS.xlsx` in the working directory with a `Results` sheet:

| File Name | Row Number | Keyword | Cell Value |
| --------- | ---------- | ------- | ---------- |

The timestamp in the filename means repeated runs never overwrite each other. If a run finds no matches, no file is written.

## Project structure

```
Cloud-Data-Aggregator/
├── go.mod
├── go.sum
├── README.md
└── src/
    ├── main.go      # Flag parsing and program entry point
    ├── auth.go      # OAuth flow, token caching, credential loading
    ├── drive.go     # Folder listing (pagination) and the concurrent worker pool
    ├── excel.go     # Downloading a file and scanning its cells for matches
    ├── results.go   # Thread-safe collection of matches
    ├── export.go    # Writing matches to an .xlsx file
    ├── utils.go     # Keyword parsing and thread-safe logging
    ├── .env.example
    └── .gitignore
```

## How it works

1. Parses the flags and compiles each keyword into a case-insensitive whole-word regex.
2. Authenticates with Google, using the cached token when available.
3. Lists every file in the folder, following page tokens until all results are retrieved.
4. Skips anything that isn't a spreadsheet, then distributes the rest across four workers.
5. Each worker downloads (or exports) its file, reads every sheet, and checks every cell against every keyword.
6. Matches are logged immediately and stored for the optional Excel export.

## Limitations

- **Subfolders are not scanned.** Only files directly inside the given folder are checked.
- **Word boundaries are ASCII-based.** Matching works as expected for English text, but keywords containing accented or non-Latin characters may not match reliably.
- **Files are downloaded fully into memory**, so very large spreadsheets use correspondingly more RAM.
- **Matched cell values are printed to the console and written to the export.** Treat both as sensitive if the scanned spreadsheets contain private data.
- **The token cache lives in the OS temp directory**, which may be cleared by the system, requiring you to sign in again.

## Security notes

- The tool requests read-only Drive access.
- The cached `token.json` is a credential: it is written with owner-only permissions (`0600`) but should still never be shared or committed.
- Keep `credentials.json`, `.env`, and any `scan-results-*.xlsx` files out of version control.

## Troubleshooting

| Problem | Likely cause |
| ------- | ------------ |
| `undefined: parseKeywords` (or similar) when running | You ran `go run main.go`. Use `go run .` instead. |
| `no credentials found` | None of the three credential sources above exist. Check the file name and working directory. |
| `Access blocked` / `403` during sign-in | Your account isn't listed as a test user on the OAuth consent screen. |
| Token refresh keeps failing | Delete `token.json` from your temp directory and sign in again. |
| `No files found in the specified folder` | Wrong folder ID, or the signed-in account can't access the folder. |
| Context deadline exceeded | A request took longer than 30 seconds. Raise `driveRequestTimeout` in `main.go` for large files. |

## Possible future improvements

- Recursive scanning of subfolders
- Configurable worker count and output path via flags
- Retry with backoff on Drive API rate limits
- Unicode-aware word-boundary matching