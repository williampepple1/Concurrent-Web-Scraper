package proxy

import (
	"math/rand"
	"net/http"
	"net/url"

	"github.com/williampepple1/concurrent-web-scraper/internal/config"
)

// Manager handles proxy configuration and rotation
type Manager struct {
	Config *config.ProxyConfig
}

// NewManager creates a new proxy manager
func NewManager(config *config.ProxyConfig) *Manager {
	return &Manager{
		Config: config,
	}
}

// GetProxyURL returns a proxy URL from the configuration
func (m *Manager) GetProxyURL() (*url.URL, error) {
	if !m.Config.Enabled || len(m.Config.List) == 0 {
		return nil, nil
	}

	// Select a proxy
	proxyStr := m.Config.List[0]
	if m.Config.Rotate && len(m.Config.List) > 1 {
		proxyStr = m.Config.List[rand.Intn(len(m.Config.List))]
	}

	// Parse the proxy URL
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	// Add authentication if provided
	if m.Config.Auth.Username != "" && m.Config.Auth.Password != "" {
		proxyURL.User = url.UserPassword(m.Config.Auth.Username, m.Config.Auth.Password)
	}

	return proxyURL, nil
}

// ApplyToTransport applies the proxy to an HTTP transport
func (m *Manager) ApplyToTransport(transport *http.Transport) (string, error) {
	proxyURL, err := m.GetProxyURL()
	if err != nil {
		return "", err
	}

	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
		return proxyURL.String(), nil
	}

	return "", nil
}
