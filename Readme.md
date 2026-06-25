# Dynamic Cloud Data Aggregator

A command-line utility built in Go that connects to the Google Drive API to scan and aggregate data from Excel spreadsheets (`.xlsx`). It allows users to dynamically input target keywords and a specific Drive folder ID to search for matching data across all spreadsheets within that folder.

## Features

* **Interactive Terminal Prompts:** Dynamically accept target keywords and Google Drive Folder IDs at runtime.
* **Case-Insensitive Scanning:** Accurately finds keyword matches regardless of capitalization (e.g., matching "Arcadium", "arcadium", or "ARCADIUM").
* **Automated Dependency Management:** Uses Go modules for easy setup.
* **Cloud Integration:** Leverages the Google Drive v3 API to locate and identify target files securely.
* **Excel Parsing:** Utilizes the robust `excelize/v2` library to read through rows and cells efficiently.

## Prerequisites

Before running this tool, you must have the following installed and configured:

1.  **Go:** [Download and install Go](https://go.dev/doc/install).
2.  **Google Cloud Console Project:**
    * Create a project in the [Google Cloud Console](https://console.cloud.google.com/).
    * Enable the **Google Drive API** for your project.
    * Create **Desktop Application Credentials** (OAuth 2.0 Client IDs).
    * Download the JSON credential file, rename it to `credentials.json`, and place it in the root directory of this project.

## Installation

1.  **Clone or create the project directory:**
    Navigate to your desired workspace and create the project folder.

2.  **Initialize the Go module:**
    ```bash
    go mod init data-aggregator
    ```

3.  **Install the required dependencies:**
    This will automatically fetch `excelize` and the Google Drive API client.
    ```bash
    go mod tidy
    ```

## Usage

1.  Run the application from your terminal:
    ```bash
    go run main.go
    ```

2.  When prompted, enter your target keywords separated by commas:
    ```text
    Enter target keywords (comma-separated, e.g., Revenue, Q1, Deficit): Melody Maker, Arcadium
    ```

3.  Next, provide the ID of the Google Drive folder you want to scan. You can find this ID in the URL of the folder when viewing it in your browser (e.g., `1aBcDeFgHiJkLmNoPqRsTuVwXyZ`).
    ```text
    Enter the Drive Folder ID to scan: [YOUR_FOLDER_ID]
    ```

4.  The script will initialize, authenticate with Google Drive, list the spreadsheets found, and print out any rows containing your target keywords.

## Credentials

Create a local `.env` file in the project root with your Google Drive credentials JSON:

```env
GOOGLE_DRIVE_CREDENTIALS={"installed":{"client_id":"...","client_secret":"..."}}
```

Do not commit this file. A sample template is available in `.env.example`.

## Dependencies

* [excelize/v2]