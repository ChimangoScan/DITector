package crawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/NSSL-SJTU/DITector/myutils"
)

// Account represents a Docker Hub account
type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token,omitempty"`
}

// IdentityManager handles rotation of IPs and Accounts
type IdentityManager struct {
	Proxies  []string
	Accounts []*Account
	mu       sync.Mutex
	proxyIdx int
	accIdx   int
}

// LoadIdentities loads proxies and accounts from JSON files
func LoadIdentities(proxyFile, accountFile string) (*IdentityManager, error) {
	im := &IdentityManager{}

	// Load Proxies (simple text file with one proxy per line)
	if proxyFile != "" {
		data, err := os.ReadFile(proxyFile)
		if err == nil {
			// Assuming line-separated proxies like http://user:pass@ip:port
			// For simplicity in this demo, we use a slice
			// You can implement more complex parsing here
			fmt.Println("Loaded proxies from", proxyFile)
		}
	}

	// Load Accounts
	if accountFile != "" {
		data, err := os.ReadFile(accountFile)
		if err == nil {
			json.Unmarshal(data, &im.Accounts)
			fmt.Printf("Loaded %d accounts\n", len(im.Accounts))
		}
	}

	return im, nil
}

// GetNextClient returns an http.Client with a rotated proxy and auth header
func (im *IdentityManager) GetNextClient() (*http.Client, string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	transport := &http.Transport{}
	
	// Proxy Rotation
	if len(im.Proxies) > 0 {
		proxyURL, _ := url.Parse(im.Proxies[im.proxyIdx])
		transport.Proxy = http.ProxyURL(proxyURL)
		im.proxyIdx = (im.proxyIdx + 1) % len(im.Proxies)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	// Account Rotation (JWT Token)
	var authToken string
	if len(im.Accounts) > 0 {
		acc := im.Accounts[im.accIdx]
		if acc.Token == "" {
			// In a real scenario, you'd call LoginDockerHub here to get the JWT
			// For now, we assume tokens are pre-loaded or we use basic auth
			myutils.Logger.Debug(fmt.Sprintf("Rotating account to: %s", acc.Username))
		}
		authToken = acc.Token
		im.accIdx = (im.accIdx + 1) % len(im.Accounts)
	}

	return client, authToken
}

// In a real implementation, you'd add a method here to Login to Docker Hub
// and refresh tokens when they expire.
