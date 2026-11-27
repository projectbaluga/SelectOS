package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	ExpiresAt int64 `json:"expires_at"`
	User struct {
		ID int64 `json:"id"`
		Username string `json:"username"`
		Role string `json:"role"`
	} `json:"user"`
}

func main() {
	if len(os.Args) < 2 { usage(); os.Exit(2) }
	switch os.Args[1] {
	case "login": onlineLogin(os.Args[2:])
	case "offline-check": offlineCheck(os.Args[2:])
	default: usage(); os.Exit(2)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "SelectOS auth client\n  login --server https://auth.example --username USER\n  offline-check --username USER\nPassword is read from SELECTOS_PASSWORD; never put passwords in shell history.\nExit status 3 means the server is unavailable (offline fallback may be attempted).")
}
func onlineLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	server := fs.String("server", "https://127.0.0.1:8088", "Authentication API base URL")
	username := fs.String("username", "", "Account username")
	cacheDir := fs.String("cache-dir", defaultCacheDir(), "Cache directory")
	_ = fs.Parse(args)

	password := os.Getenv("SELECTOS_PASSWORD")
	if *username == "" || len(*username) > 64 || password == "" || len(password) > 256 {
		fmt.Fprintln(os.Stderr, "Set a valid --username and SELECTOS_PASSWORD (maximum password length: 256 bytes).")
		os.Exit(2)
	}
	base, err := url.Parse(*server)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		fmt.Fprintln(os.Stderr, "server must be a base http(s) URL without credentials, path, query, or fragment")
		os.Exit(2)
	}
	if base.Scheme != "https" && os.Getenv("SELECTOS_AUTH_ALLOW_HTTP") != "1" {
		fmt.Fprintln(os.Stderr, "HTTPS is required. For isolated-LAN testing only, set SELECTOS_AUTH_ALLOW_HTTP=1.")
		os.Exit(2)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/v1/auth/login"
	body, err := json.Marshal(loginRequest{Username: *username, Password: password})
	if err != nil { fatal(fmt.Errorf("could not prepare login request")) }
	req, err := http.NewRequest(http.MethodPost, base.String(), bytes.NewReader(body))
	if err != nil { fatal(fmt.Errorf("could not prepare login request")) }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Authentication server unavailable:", err)
		os.Exit(3) // Only this status permits the Gate hook to try offline verification.
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil { fmt.Fprintln(os.Stderr, "Could not read auth server response."); os.Exit(3) }
	if resp.StatusCode >= 500 {
		fmt.Fprintf(os.Stderr, "Authentication service unavailable (HTTP %d).\n", resp.StatusCode)
		os.Exit(3)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Server rejected login (HTTP %d). Offline fallback is not permitted for rejected credentials.\n", resp.StatusCode)
		os.Exit(1)
	}
	var result loginResponse
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&result); err != nil || result.AccessToken == "" || !strings.EqualFold(result.TokenType, "Bearer") ||
		result.User.ID <= 0 || result.User.Username == "" || !strings.EqualFold(result.User.Username, *username) ||
		result.ExpiresAt <= time.Now().Unix() {
		fatal(fmt.Errorf("auth server returned an invalid or expired login response"))
	}
	if err = SaveOfflineCache(*cacheDir, result.User.Username, password, *server, result.User.Role, 72*time.Hour); err != nil {
		fatal(fmt.Errorf("online login succeeded, but offline cache could not be saved: %w", err))
	}
	fmt.Printf("Online login successful: %s (%s). Offline cache valid for up to 72 hours on this device.\n", result.User.Username, result.User.Role)
}
func offlineCheck(args []string) {
	fs := flag.NewFlagSet("offline-check", flag.ExitOnError)
	username := fs.String("username", "", "Account username")
	cacheDir := fs.String("cache-dir", defaultCacheDir(), "Cache directory")
	_ = fs.Parse(args)
	password := os.Getenv("SELECTOS_PASSWORD")
	if *username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "Set --username and SELECTOS_PASSWORD.")
		os.Exit(2)
	}
	role, err := VerifyOfflineCache(*cacheDir, *username, password)
	if err != nil { fmt.Fprintln(os.Stderr, "Offline login denied:", err); os.Exit(1) }
	fmt.Printf("Offline login accepted: %s (%s). Server policy changes may not be known until reconnect.\n", *username, role)
}
func defaultCacheDir() string {
	if v := os.Getenv("SELECTOS_AUTH_CACHE"); v != "" { return v }
	return "/var/lib/selectos-auth-client"
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
