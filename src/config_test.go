package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	content := "\ufeff# comment\n" +
		"WEBVPN_USER=\"3250103873\"\n" +
		"WEBVPN_PASS='vpn-pass'\n" +
		"CC98_ACCOUNT_COUNT=2\n" +
		"CC98_USER_1=anchorite\n" +
		"CC98_PASS_1='p@ss1'\n" +
		"CC98_USER_2=\"副江凡\"\n" +
		"CC98_PASS_2=\"Fjf123456\"\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg, err := LoadConfig(envPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.WebVPNUser != "3250103873" {
		t.Fatalf("unexpected webvpn user: %s", cfg.WebVPNUser)
	}
	if len(cfg.Accounts) != 2 {
		t.Fatalf("unexpected account count: %d", len(cfg.Accounts))
	}
	if cfg.Accounts[1].Username != "副江凡" {
		t.Fatalf("unexpected second username: %s", cfg.Accounts[1].Username)
	}
}

func TestLoadConfigCollectsMissingFields(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	content := "WEBVPN_USER=\nWEBVPN_PASS=\nCC98_ACCOUNT_COUNT=2\nCC98_USER_1=\nCC98_PASS_1=pass\nCC98_USER_2=user2\nCC98_PASS_2=\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	_, err := LoadConfig(envPath)
	var validationErr *ConfigValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}

	expected := []string{"CC98_PASS_2", "CC98_USER_1", "WEBVPN_PASS", "WEBVPN_USER"}
	if !slices.Equal(validationErr.MissingFields, expected) {
		t.Fatalf("unexpected missing fields: %#v", validationErr.MissingFields)
	}
}

func TestEnsureEnvFileCreatesTemplate(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	created, err := EnsureEnvFile(envPath)
	if err != nil {
		t.Fatalf("EnsureEnvFile returned error: %v", err)
	}
	if !created {
		t.Fatal("expected template to be created")
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if string(content) != defaultEnvTemplate {
		t.Fatalf("unexpected template content: %q", string(content))
	}
}
