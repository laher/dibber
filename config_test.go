package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to set up a temp config file for testing
func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	// Save original HOME
	origHome := os.Getenv("HOME")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dibber-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Set HOME to temp directory
	_ = os.Setenv("HOME", tmpDir)

	cleanup := func() {
		_ = os.Setenv("HOME", origHome)
		_ = os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestLoadConfigNotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	_, err := LoadConfig()
	if err != ErrConfigNotFound {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		Salt:             "dGVzdC1zYWx0", // base64 of "test-salt"
		EncryptedDataKey: "encrypted-key-here",
		Connections: map[string]*Connection{
			"prod": {
				EncryptedDSN: "encrypted-dsn-1",
				Type:         "postgres",
			},
			"dev": {
				EncryptedDSN: "encrypted-dsn-2",
				Type:         "mysql",
			},
		},
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Salt != cfg.Salt {
		t.Errorf("Salt = %q, want %q", loaded.Salt, cfg.Salt)
	}
	if loaded.EncryptedDataKey != cfg.EncryptedDataKey {
		t.Errorf("EncryptedDataKey = %q, want %q", loaded.EncryptedDataKey, cfg.EncryptedDataKey)
	}
	if len(loaded.Connections) != 2 {
		t.Errorf("len(Connections) = %d, want 2", len(loaded.Connections))
	}
	if loaded.Connections["prod"].Type != "postgres" {
		t.Errorf("prod type = %q, want postgres", loaded.Connections["prod"].Type)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	tmpDir, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		Salt:             "test",
		EncryptedDataKey: "test",
		Connections:      make(map[string]*Connection),
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	path := filepath.Join(tmpDir, configFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// Check permissions are 0600 (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestConfigHasVault(t *testing.T) {
	cfg := &Config{
		Connections: make(map[string]*Connection),
	}

	if cfg.HasVault() {
		t.Error("empty config should not have vault")
	}

	cfg.Salt = "test-salt"
	if cfg.HasVault() {
		t.Error("config with only salt should not have vault")
	}

	cfg.EncryptedDataKey = "test-key"
	if !cfg.HasVault() {
		t.Error("config with salt and key should have vault")
	}
}

func TestConfigConnectionNames(t *testing.T) {
	cfg := &Config{
		Connections: map[string]*Connection{
			"zebra": {},
			"apple": {},
			"mango": {},
		},
	}

	names := cfg.ConnectionNames()
	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}
	// Should be sorted
	if names[0] != "apple" || names[1] != "mango" || names[2] != "zebra" {
		t.Errorf("names should be sorted: got %v", names)
	}
}

func TestVaultManagerIntegration(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	vm := NewVaultManager()
	err := vm.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if vm.HasVault() {
		t.Error("new vault manager should not have vault")
	}

	// Initialize with password
	password := "test-password-123"
	if err := vm.InitializeWithPassword(password); err != nil {
		t.Fatalf("InitializeWithPassword failed: %v", err)
	}

	if !vm.HasVault() {
		t.Error("should have vault after initialization")
	}
	if !vm.IsUnlocked() {
		t.Error("should be unlocked after initialization")
	}

	// Add a connection
	if err := vm.AddConnection("prod", "postgres://localhost/prod", "postgres", ""); err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Verify connection can be retrieved
	dsn, dbType, _, err := vm.GetConnection("prod")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}
	if dsn != "postgres://localhost/prod" {
		t.Errorf("dsn = %q, want postgres://localhost/prod", dsn)
	}
	if dbType != "postgres" {
		t.Errorf("dbType = %q, want postgres", dbType)
	}

	// Lock the vault
	vm.Lock()
	if vm.IsUnlocked() {
		t.Error("should be locked after Lock()")
	}

	// Can't get connection when locked
	_, _, _, err = vm.GetConnection("prod")
	if err != ErrVaultLocked {
		t.Errorf("expected ErrVaultLocked, got %v", err)
	}

	// Create new vault manager and unlock
	vm2 := NewVaultManager()
	if err := vm2.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if err := vm2.Unlock(password); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Should be able to get the connection
	dsn, _, _, err = vm2.GetConnection("prod")
	if err != nil {
		t.Fatalf("GetConnection failed after reload: %v", err)
	}
	if dsn != "postgres://localhost/prod" {
		t.Errorf("dsn = %q after reload, want postgres://localhost/prod", dsn)
	}

	// Wrong password should fail
	vm3 := NewVaultManager()
	_ = vm3.LoadConfig()
	if err := vm3.Unlock("wrong-password"); err == nil {
		t.Error("unlock with wrong password should fail")
	}
}

func TestVaultManagerRemoveConnection(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	vm := NewVaultManager()
	_ = vm.LoadConfig()
	_ = vm.InitializeWithPassword("test-password")
	_ = vm.AddConnection("conn1", "dsn1", "", "")
	_ = vm.AddConnection("conn2", "dsn2", "", "")

	names := vm.ListConnections()
	if len(names) != 2 {
		t.Errorf("should have 2 connections, got %d", len(names))
	}

	if err := vm.RemoveConnection("conn1"); err != nil {
		t.Fatalf("RemoveConnection failed: %v", err)
	}

	names = vm.ListConnections()
	if len(names) != 1 {
		t.Errorf("should have 1 connection after removal, got %d", len(names))
	}
	if names[0] != "conn2" {
		t.Errorf("remaining connection should be conn2, got %s", names[0])
	}

	// Remove non-existent should fail
	err := vm.RemoveConnection("nonexistent")
	if err != ErrConnectionNotFound {
		t.Errorf("expected ErrConnectionNotFound, got %v", err)
	}
}

func TestVaultManagerChangePassword(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	vm := NewVaultManager()
	_ = vm.LoadConfig()
	oldPassword := "old-password"
	newPassword := "new-password"

	_ = vm.InitializeWithPassword(oldPassword)
	_ = vm.AddConnection("test", "test-dsn", "", "")

	// Change password
	if err := vm.ChangePassword(newPassword); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// Lock and try new password
	vm.Lock()

	// Old password should fail
	vm2 := NewVaultManager()
	_ = vm2.LoadConfig()
	if err := vm2.Unlock(oldPassword); err == nil {
		t.Error("old password should no longer work")
	}

	// New password should work
	vm3 := NewVaultManager()
	_ = vm3.LoadConfig()
	if err := vm3.Unlock(newPassword); err != nil {
		t.Fatalf("new password should work: %v", err)
	}

	// Data should still be accessible
	dsn, _, _, err := vm3.GetConnection("test")
	if err != nil {
		t.Fatalf("GetConnection failed after password change: %v", err)
	}
	if dsn != "test-dsn" {
		t.Errorf("dsn = %q, want test-dsn", dsn)
	}
}
