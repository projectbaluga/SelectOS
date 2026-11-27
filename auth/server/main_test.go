package main

import (
	"strings"
	"testing"
	"time"
)

func TestSignedTokenRoundTrip(t *testing.T) {
	s := &Server{tokenKey: []byte(strings.Repeat("k", 32))}
	want := claims{UserID: 42, Username: "student01", Role: "user", Expires: time.Now().Add(time.Hour).Unix()}
	token, err := s.sign(want)
	if err != nil { t.Fatal(err) }
	got, err := s.verify(token)
	if err != nil { t.Fatal(err) }
	if got.UserID != want.UserID || got.Username != want.Username || got.Role != want.Role || got.Expires != want.Expires {
		t.Fatalf("claims mismatch: got %#v, want %#v", got, want)
	}
}

func TestSignedTokenRejectsTampering(t *testing.T) {
	s := &Server{tokenKey: []byte(strings.Repeat("k", 32))}
	token, err := s.sign(claims{UserID: 42, Username: "student01", Role: "user", Expires: time.Now().Add(time.Hour).Unix()})
	if err != nil { t.Fatal(err) }
	if _, err := s.verify(token + "x"); err == nil { t.Fatal("expected tampered token to be rejected") }
}

func TestSignedTokenRejectsExpiredAndExcessiveExpiry(t *testing.T) {
	s := &Server{tokenKey: []byte(strings.Repeat("k", 32))}
	for _, expiry := range []int64{time.Now().Add(-time.Minute).Unix(), time.Now().Add(9*time.Hour).Unix()} {
		token, err := s.sign(claims{UserID: 42, Username: "student01", Role: "user", Expires: expiry})
		if err != nil { t.Fatal(err) }
		if _, err := s.verify(token); err == nil { t.Fatalf("expected expiry %d to be rejected", expiry) }
	}
}
