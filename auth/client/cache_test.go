package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOfflineCacheRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := SaveOfflineCache(dir, "student01", "correct horse battery staple", "https://auth.example", "user", time.Hour); err != nil {
		t.Fatal(err)
	}
	role, err := VerifyOfflineCache(dir, "student01", "correct horse battery staple")
	if err != nil { t.Fatal(err) }
	if role != "user" { t.Fatalf("role = %q, want user", role) }
}

func TestOfflineCacheRejectsWrongCredentials(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := SaveOfflineCache(dir, "student01", "correct horse battery staple", "https://auth.example", "user", time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyOfflineCache(dir, "otheruser", "correct horse battery staple"); err == nil {
		t.Fatal("expected username mismatch to be rejected")
	}
	if _, err := VerifyOfflineCache(dir, "student01", "wrong password"); err == nil {
		t.Fatal("expected wrong password to be rejected")
	}
}

func TestOfflineCacheExpires(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := SaveOfflineCache(dir, "student01", "correct horse battery staple", "https://auth.example", "user", time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := VerifyOfflineCache(dir, "student01", "correct horse battery staple"); err == nil {
		t.Fatal("expected expired cache to be rejected")
	}
}

func TestOfflineCacheRejectsUnsafePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := SaveOfflineCache(dir, "student01", "correct horse battery staple", "https://auth.example", "user", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, keyFile), 0644); err != nil { t.Fatal(err) }
	if _, err := VerifyOfflineCache(dir, "student01", "correct horse battery staple"); err == nil {
		t.Fatal("expected world-readable device key to be rejected")
	}
}

func TestOfflineCacheRejectsExcessiveLifetime(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	if err := SaveOfflineCache(dir, "student01", "correct horse battery staple", "https://auth.example", "user", 73*time.Hour); err == nil {
		t.Fatal("expected lifetime above 72 hours to be rejected")
	}
}
