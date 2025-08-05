# Concurrent Web Scraper

A powerful, concurrent web scraper built in Go that leverages goroutines and channels for efficient parallel processing. This tool can scrape multiple websites simultaneously, extract data using various methods, and handle JavaScript-rendered pages.

## Features

- **Concurrent Scraping**: Uses goroutines and channels for parallel processing
- **Rate Limiting**: Prevents overloading target servers
- **Retry Logic**: Automatically retries failed requests with exponential backoff
- **User Agent Rotation**: Rotates between different user agents to avoid detection
- **Proxy Support**: Can use HTTP/HTTPS proxies with authentication
- **Data Extraction**: Extracts data using CSS selectors, XPath, and regular expressions
- **JavaScript Rendering**: Supports scraping JavaScript-rendered pages using headless Chrome
- **Configurable**: Supports YAML configuration files and command-line flags
- **Output Options**: Saves results in JSON format (CSV coming soon)

## Installation

### Prerequisites

- Go 1.16 or higher
- Chrome/Chromium (only if using JavaScript rendering)

### Installing

```bash
# Clone the repository
git clone https://github.com/yourusername/concurrent-web-scraper.git
cd concurrent-web-scraper

# Install dependencies
go mod download
```

## Usage

### Basic Usage

```bash
# Run with default settings
go run main.go

# Run with command-line options
go run main.go -workers 5 -rate-limit 2s -input urls.txt -output results.json
```

### Using a Configuration File

```bash
# Run with a configuration file
go run main.go -config config.yaml
```

### Command-Line Options

- `-config`: Path to configuration file (YAML)
- `-input`: File containing URLs to scrape (one per line)
- `-output`: File to save results to
- `-workers`: Number of concurrent workers
- `-rate-limit`: Delay between requests
- `-retries`: Maximum number of retries per URL
- `-retry-delay`: Base delay between retries
- `-title-selector`: CSS selector for title extraction
- `-heading-selector`: CSS selector for heading extraction
- `-proxy`: Enable proxy support
- `-browser`: Enable browser-based scraping

## Configuration File

The scraper can be configured using a YAML file. Here's an example:

```yaml
# Scraping Settings
scraper:
  workers: 5                   # Number of concurrent workers
  rate_limit: 2s               # Delay between requests
  max_retries: 3               # Maximum number of retries per URL
  retry_delay: 2s              # Base delay between retries
  timeout: 30s                 # Request timeout

# Input/Output Settings
io:
  input_file: "urls.txt"       # File containing URLs to scrape
  output_file: "results.json"  # File to save results to
  output_format: "json"        # Output format (json, csv)

# Data Extraction Settings
extraction:
  selectors:                   # CSS selectors for data extraction
    title: "title"             # Page title
    heading: "h1"              # Main heading
  xpath:                       # XPath selectors for data extraction
    price: "//span[@class='price']"
  regex:                       # Regular expressions for data extraction
    email: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"

# Proxy Settings
proxies:
  enabled: false               # Enable proxy support
  rotate: true                 # Rotate between proxies
  list:                        # List of proxies to use
    - http://proxy1.example.com:8080

# Browser Settings (for JavaScript rendering)
browser:
  enabled: false               # Enable browser-based scraping
  headless: true               # Run browser in headless mode
  wait_time: 5s                # Time to wait for JavaScript to execute
  screenshot: false            # Take screenshots of rendered pages
```

## Examples

### Scraping with JavaScript Rendering

```bash
go run main.go -browser -input spa-websites.txt -output spa-results.json
```

### Using Proxies

Create a configuration file with proxy settings and run:

```bash
go run main.go -config proxy-config.yaml
```

### Extracting Specific Data

```bash
go run main.go -title-selector "h1.main-title" -heading-selector "div.content h2"
```

## Building

To build a standalone executable:

```bash
go build -o scraper
```

Then run it:

```bash
./scraper -config config.yaml
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.