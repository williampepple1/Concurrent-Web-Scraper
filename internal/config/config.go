package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig holds the complete application configuration
type AppConfig struct {
	Scraper    ScraperConfig    `yaml:"scraper"`
	IO         IOConfig         `yaml:"io"`
	Extraction ExtractionConfig `yaml:"extraction"`
	Proxies    ProxyConfig      `yaml:"proxies"`
	Browser    BrowserConfig    `yaml:"browser"`
}

// ScraperConfig holds the scraper configuration
type ScraperConfig struct {
	Workers    int           `yaml:"workers"`
	RateLimit  time.Duration `yaml:"rate_limit"`
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
	Timeout    time.Duration `yaml:"timeout"`
	UserAgents []string      `yaml:"user_agents,omitempty"`
}

// IOConfig holds the input/output configuration
type IOConfig struct {
	InputFile    string `yaml:"input_file"`
	OutputFile   string `yaml:"output_file"`
	OutputFormat string `yaml:"output_format"`
}

// ExtractionConfig holds the data extraction configuration
type ExtractionConfig struct {
	Selectors map[string]string `yaml:"selectors"`
	XPath     map[string]string `yaml:"xpath"`
	Regex     map[string]string `yaml:"regex"`
}

// ProxyConfig holds the proxy configuration
type ProxyConfig struct {
	Enabled bool     `yaml:"enabled"`
	Rotate  bool     `yaml:"rotate"`
	List    []string `yaml:"list"`
	Auth    struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`
}

// BrowserConfig holds the browser configuration for JavaScript rendering
type BrowserConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Headless      bool          `yaml:"headless"`
	UserAgent     string        `yaml:"user_agent"`
	WaitTime      time.Duration `yaml:"wait_time"`
	Screenshot    bool          `yaml:"screenshot"`
	ScreenshotDir string        `yaml:"screenshot_dir"`
}

// Load loads the configuration from a YAML file
func Load(filename string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set default user agents if none provided
	if len(config.Scraper.UserAgents) == 0 {
		config.Scraper.UserAgents = DefaultUserAgents
	}

	return &config, nil
}

// CreateDefault creates a default configuration
func CreateDefault(numWorkers int, rateLimitDelay, retryDelay time.Duration, maxRetries int,
	inputFile, outputFile, titleSelector, headingSelector string, enableProxy, enableBrowser bool) *AppConfig {
	return &AppConfig{
		Scraper: ScraperConfig{
			Workers:    numWorkers,
			RateLimit:  rateLimitDelay,
			MaxRetries: maxRetries,
			RetryDelay: retryDelay,
			Timeout:    30 * time.Second,
			UserAgents: DefaultUserAgents,
		},
		IO: IOConfig{
			InputFile:    inputFile,
			OutputFile:   outputFile,
			OutputFormat: "json",
		},
		Extraction: ExtractionConfig{
			Selectors: map[string]string{
				"title":   titleSelector,
				"heading": headingSelector,
			},
			XPath: map[string]string{},
			Regex: map[string]string{},
		},
		Proxies: ProxyConfig{
			Enabled: enableProxy,
			Rotate:  true,
			List:    []string{},
		},
		Browser: BrowserConfig{
			Enabled:       enableBrowser,
			Headless:      true,
			UserAgent:     DefaultUserAgents[0],
			WaitTime:      5 * time.Second,
			Screenshot:    false,
			ScreenshotDir: "screenshots",
		},
	}
}
