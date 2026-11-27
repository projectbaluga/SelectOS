package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const cacheFile = "offline-cache.enc"
const keyFile = "device.key"
const maxCacheLifetime = 72 * time.Hour
const cacheAAD = "SelectOS-offline-cache-v1"

type offlineRecord struct {
	Username string `json:"username"`
	Role string `json:"role"`
	Server string `json:"server"`
	Salt []byte `json:"salt"`
	Verifier []byte `json:"verifier"`
	CreatedAt int64 `json:"created_at"`
	ExpiresAt int64 `json:"expires_at"`
}

// SaveOfflineCache must only be called after a successful online login.
// The directory and key are device-local secrets; root ownership is recommended.
func SaveOfflineCache(dir, username, password, server, role string, lifetime time.Duration) error {
	if strings.TrimSpace(username) == "" || len(username) > 64 || password == "" || len(password) > 256 {
		return errors.New("invalid cache credentials")
	}
	if lifetime <= 0 || lifetime > maxCacheLifetime {
		return errors.New("offline cache lifetime must be between 0 and 72 hours")
	}
	if err := os.MkdirAll(dir, 0700); err != nil { return err }
	if err := os.Chmod(dir, 0700); err != nil { return err }
	info, err := os.Lstat(dir)
	if err != nil { return err }
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 { return errors.New("cache path must be a real directory") }

	keyPath := filepath.Join(dir, keyFile)
	key, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		candidate := make([]byte, 32)
		if _, err = rand.Read(candidate); err != nil { return err }
		f, createErr := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if createErr == nil {
			if _, err = f.Write(candidate); err == nil { err = f.Sync() }
			closeErr := f.Close()
			if err == nil { err = closeErr }
			if err != nil { _ = os.Remove(keyPath); return err }
			key = candidate
		} else if os.IsExist(createErr) {
			key, err = os.ReadFile(keyPath)
		} else { return createErr }
	} else if err != nil { return err }
	if len(key) != 32 { return errors.New("invalid device key; remove cache and re-enroll this device") }
	keyInfo, err := os.Lstat(keyPath)
	if err != nil { return err }
	if !keyInfo.Mode().IsRegular() || keyInfo.Mode().Perm()&0077 != 0 { return errors.New("device key must be a regular file with mode 0600") }

	salt := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil { return err }
	verifier := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	now := time.Now()
	record := offlineRecord{Username: strings.TrimSpace(username), Role: role, Server: server, Salt: salt, Verifier: verifier, CreatedAt: now.Unix(), ExpiresAt: now.Add(lifetime).Unix()}
	plain, err := json.Marshal(record); if err != nil { return err }
	block, err := aes.NewCipher(key); if err != nil { return err }
	gcm, err := cipher.NewGCM(block); if err != nil { return err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil { return err }
	sealed := gcm.Seal(nonce, nonce, plain, []byte(cacheAAD))

	tmp, err := os.CreateTemp(dir, ".offline-cache-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil { _, err = tmp.Write(sealed) }
	if err == nil { err = tmp.Sync() }
	closeErr := tmp.Close()
	if err == nil { err = closeErr }
	if err != nil { return err }
	if err = os.Rename(tmpName, filepath.Join(dir, cacheFile)); err != nil { return err }
	d, err := os.Open(dir); if err == nil { err = d.Sync(); _ = d.Close() }
	return err
}

func VerifyOfflineCache(dir, username, password string) (string, error) {
	if strings.TrimSpace(username) == "" || password == "" || len(password) > 256 { return "", errors.New("invalid credentials") }
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return "", errors.New("offline cache directory is missing or permissions are unsafe")
	}
	keyPath := filepath.Join(dir, keyFile)
	keyInfo, err := os.Lstat(keyPath)
	if err != nil || !keyInfo.Mode().IsRegular() || keyInfo.Mode().Perm()&0077 != 0 {
		return "", errors.New("device key missing or permissions are unsafe")
	}
	key, err := os.ReadFile(keyPath)
	if err != nil || len(key) != 32 { return "", errors.New("invalid device key") }
	cachePath := filepath.Join(dir, cacheFile)
	cacheInfo, err := os.Lstat(cachePath)
	if err != nil || !cacheInfo.Mode().IsRegular() || cacheInfo.Mode().Perm()&0077 != 0 {
		return "", errors.New("offline cache missing or permissions are unsafe")
	}
	sealed, err := os.ReadFile(cachePath)
	if err != nil { return "", errors.New("offline cache unavailable") }
	block, err := aes.NewCipher(key); if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block); if err != nil { return "", err }
	if len(sealed) < gcm.NonceSize()+gcm.Overhead() { return "", errors.New("corrupt offline cache") }
	nonce := sealed[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, sealed[gcm.NonceSize():], []byte(cacheAAD))
	if err != nil { return "", errors.New("offline cache authentication failed") }
	var record offlineRecord
	if err = json.Unmarshal(plain, &record); err != nil { return "", errors.New("corrupt offline cache record") }
	if record.Username != strings.TrimSpace(username) { return "", errors.New("username does not match this device cache") }
	now := time.Now().Unix()
	if record.CreatedAt <= 0 || record.ExpiresAt <= record.CreatedAt || record.ExpiresAt-record.CreatedAt > int64(maxCacheLifetime.Seconds()) || now > record.ExpiresAt {
		return "", errors.New("offline cache expired or has invalid timestamps; reconnect to the LAN server")
	}
	if len(record.Salt) != 16 || len(record.Verifier) != 32 { return "", fmt.Errorf("invalid offline verifier") }
	candidate := argon2.IDKey([]byte(password), record.Salt, 3, 64*1024, 2, 32)
	if subtle.ConstantTimeCompare(candidate, record.Verifier) != 1 { return "", errors.New("incorrect password") }
	return record.Role, nil
}
