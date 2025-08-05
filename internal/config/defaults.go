package config

// DefaultUserAgents provides a list of common user agents
var DefaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
}

// DefaultURLs provides a list of default URLs to scrape if none are provided
var DefaultURLs = []string{
	"https://www.google.com",
	"https://www.github.com",
	"https://www.golang.org",
	"https://www.wikipedia.org",
	"https://www.reddit.com",
}
