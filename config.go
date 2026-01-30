package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const configFileName = ".dibber.yaml"

var (
	ErrConfigNotFound     = errors.New("config file not found")
	ErrVaultNotConfigured = errors.New("vault not configured - run with -add-conn first")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrVaultLocked        = errors.New("vault is locked")
)

// Connection represents an encrypted connection entry
type Connection struct {
	EncryptedDSN string `yaml:"encrypted_dsn"`
	Type         string `yaml:"type,omitempty"`  // mysql, postgres, sqlite (optional, for auto-detection override)
	Theme        string `yaml:"theme,omitempty"` // optional theme name for visual distinction
}

// Config represents the ~/.dibber.yaml configuration file
type Config struct {
	// Salt for the KDF (base64 encoded)
	Salt string `yaml:"salt"`

	// Encrypted data key (base64 encoded, encrypted with master-derived key)
	EncryptedDataKey string `yaml:"encrypted_data_key"`

	// Connections keyed by name
	Connections map[string]*Connection `yaml:"connections"`
}

// configPath returns the full path to the config file
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

// LoadConfig loads the config from ~/.dibber.yaml
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Connections == nil {
		cfg.Connections = make(map[string]*Connection)
	}

	return &cfg, nil
}

// SaveConfig saves the config to ~/.dibber.yaml
func SaveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetSalt returns the decoded salt from the config
func (c *Config) GetSalt() ([]byte, error) {
	if c.Salt == "" {
		return nil, ErrVaultNotConfigured
	}
	salt, err := base64.StdEncoding.DecodeString(c.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	return salt, nil
}

// SetSalt sets the salt (base64 encodes it)
func (c *Config) SetSalt(salt []byte) {
	c.Salt = base64.StdEncoding.EncodeToString(salt)
}

// HasVault returns true if the vault has been initialized
func (c *Config) HasVault() bool {
	return c.Salt != "" && c.EncryptedDataKey != ""
}

// ConnectionNames returns a sorted list of connection names
func (c *Config) ConnectionNames() []string {
	names := make([]string, 0, len(c.Connections))
	for name := range c.Connections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasConnection returns true if a connection with the given name exists
func (c *Config) HasConnection(name string) bool {
	_, ok := c.Connections[name]
	return ok
}

// VaultManager manages the vault state and config
type VaultManager struct {
	config *Config
	vault  *Vault
}

// NewVaultManager creates a new vault manager
func NewVaultManager() *VaultManager {
	return &VaultManager{
		vault: NewVault(),
	}
}

// LoadConfig loads the configuration file
func (vm *VaultManager) LoadConfig() error {
	cfg, err := LoadConfig()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			// Create a new empty config
			vm.config = &Config{
				Connections: make(map[string]*Connection),
			}
			return nil
		}
		return err
	}
	vm.config = cfg
	return nil
}

// HasVault returns true if the vault has been initialized
func (vm *VaultManager) HasVault() bool {
	return vm.config != nil && vm.config.HasVault()
}

// IsUnlocked returns true if the vault is currently unlocked
func (vm *VaultManager) IsUnlocked() bool {
	return vm.vault.IsUnlocked()
}

// Unlock unlocks the vault with the master password
func (vm *VaultManager) Unlock(password string) error {
	if vm.config == nil {
		return ErrVaultNotConfigured
	}

	salt, err := vm.config.GetSalt()
	if err != nil {
		return err
	}

	dataKey, err := UnlockVault(password, salt, vm.config.EncryptedDataKey)
	if err != nil {
		return err
	}

	vm.vault.dataKey = dataKey
	vm.vault.isUnlocked = true

	// Decrypt all connections into memory
	for name, conn := range vm.config.Connections {
		dsn, err := DecryptDSN(dataKey, conn.EncryptedDSN)
		if err != nil {
			// If one fails, lock and return error
			vm.vault.Lock()
			return fmt.Errorf("failed to decrypt connection %q: %w", name, err)
		}
		vm.vault.connections[name] = dsn
	}

	return nil
}

// Lock locks the vault
func (vm *VaultManager) Lock() {
	vm.vault.Lock()
}

// InitializeWithPassword initializes a new vault with the given password
func (vm *VaultManager) InitializeWithPassword(password string) error {
	salt, encryptedDataKey, dataKey, err := InitializeVault(password)
	if err != nil {
		return err
	}

	if vm.config == nil {
		vm.config = &Config{
			Connections: make(map[string]*Connection),
		}
	}

	vm.config.SetSalt(salt)
	vm.config.EncryptedDataKey = encryptedDataKey

	// Keep vault unlocked with the data key
	vm.vault.dataKey = dataKey
	vm.vault.isUnlocked = true

	return SaveConfig(vm.config)
}

// AddConnection adds a new encrypted connection
func (vm *VaultManager) AddConnection(name, dsn, dbType, theme string) error {
	if !vm.vault.IsUnlocked() {
		return ErrVaultLocked
	}

	// Encrypt the DSN
	encryptedDSN, err := EncryptDSN(vm.vault.dataKey, dsn)
	if err != nil {
		return fmt.Errorf("failed to encrypt DSN: %w", err)
	}

	// Add to config
	vm.config.Connections[name] = &Connection{
		EncryptedDSN: encryptedDSN,
		Type:         dbType,
		Theme:        theme,
	}

	// Add to in-memory vault
	vm.vault.connections[name] = dsn

	// Save config
	return SaveConfig(vm.config)
}

// RemoveConnection removes a connection
func (vm *VaultManager) RemoveConnection(name string) error {
	if !vm.vault.IsUnlocked() {
		return ErrVaultLocked
	}

	if !vm.config.HasConnection(name) {
		return ErrConnectionNotFound
	}

	delete(vm.config.Connections, name)
	delete(vm.vault.connections, name)

	return SaveConfig(vm.config)
}

// GetConnection returns a decrypted connection DSN, type, and theme
func (vm *VaultManager) GetConnection(name string) (dsn string, dbType string, theme string, err error) {
	if !vm.vault.IsUnlocked() {
		return "", "", "", ErrVaultLocked
	}

	dsn, ok := vm.vault.connections[name]
	if !ok {
		return "", "", "", ErrConnectionNotFound
	}

	// Get the type and theme if specified
	if conn, ok := vm.config.Connections[name]; ok {
		dbType = conn.Type
		theme = conn.Theme
	}

	return dsn, dbType, theme, nil
}

// ListConnections returns a list of connection names
func (vm *VaultManager) ListConnections() []string {
	if vm.config == nil {
		return nil
	}
	return vm.config.ConnectionNames()
}

// ChangePassword changes the master password (re-encrypts data key)
func (vm *VaultManager) ChangePassword(newPassword string) error {
	if !vm.vault.IsUnlocked() {
		return ErrVaultLocked
	}

	// Generate new salt
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	// Derive new key
	derivedKey := DeriveKey(newPassword, salt)

	// Re-encrypt the existing data key with the new derived key
	encryptedDataKey, err := EncryptDataKey(derivedKey, vm.vault.dataKey)
	if err != nil {
		return err
	}

	// Update config
	vm.config.SetSalt(salt)
	vm.config.EncryptedDataKey = encryptedDataKey

	return SaveConfig(vm.config)
}
