package models

import (
	"time"
)

// Result represents the result of scraping a URL
type Result struct {
	URL        string                 `json:"url"`
	Content    string                 `json:"content,omitempty"`
	Extracted  map[string]interface{} `json:"extracted,omitempty"`
	Err        string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Retries    int                    `json:"retries"`
	StatusCode int                    `json:"status_code,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Screenshot string                 `json:"screenshot,omitempty"`
	JSRendered bool                   `json:"js_rendered,omitempty"`
	ProxyUsed  string                 `json:"proxy_used,omitempty"`
}
