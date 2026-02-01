package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended for 2023+)
const (
	argonTime    = 3         // Number of iterations
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4         // Parallelism
	argonKeyLen  = 32        // 256-bit key for AES-256
	saltLen      = 16        // 128-bit salt
	nonceLen     = 12        // 96-bit nonce for GCM
	dataKeyLen   = 32        // 256-bit data key
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: invalid password or corrupted data")
	ErrInvalidData      = errors.New("invalid encrypted data format")
)

// Vault holds the in-memory decrypted data key and connection data
type Vault struct {
	dataKey     []byte            // Decrypted data key, kept in memory while unlocked
	connections map[string]string // Decrypted connection DSNs, keyed by name
	isUnlocked  bool
}

// NewVault creates a new empty vault
func NewVault() *Vault {
	return &Vault{
		connections: make(map[string]string),
		isUnlocked:  false,
	}
}

// IsUnlocked returns whether the vault is currently unlocked
func (v *Vault) IsUnlocked() bool {
	return v.isUnlocked
}

// Lock clears the data key and locks the vault
func (v *Vault) Lock() {
	// Clear the data key from memory (best effort - Go doesn't guarantee this)
	if v.dataKey != nil {
		for i := range v.dataKey {
			v.dataKey[i] = 0
		}
		v.dataKey = nil
	}
	// Clear connection data
	for k := range v.connections {
		delete(v.connections, k)
	}
	v.isUnlocked = false
}

// GetConnection returns a decrypted DSN by name
func (v *Vault) GetConnection(name string) (string, bool) {
	dsn, ok := v.connections[name]
	return dsn, ok
}

// ListConnections returns all connection names
func (v *Vault) ListConnections() []string {
	names := make([]string, 0, len(v.connections))
	for name := range v.connections {
		names = append(names, name)
	}
	return names
}

// GenerateSalt generates a cryptographically random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateDataKey generates a cryptographically random data key
func GenerateDataKey() ([]byte, error) {
	key := make([]byte, dataKeyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}
	return key, nil
}

// DeriveKey derives an encryption key from a password using Argon2id
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce
// Returns base64-encoded ciphertext (nonce prepended to ciphertext)
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext to the nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext (with prepended nonce) using AES-256-GCM
func Decrypt(key []byte, ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(ciphertext) < nonceLen {
		return nil, ErrInvalidData
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:nonceLen]
	ciphertext = ciphertext[nonceLen:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptDataKey encrypts the data key with the derived master key
func EncryptDataKey(derivedKey, dataKey []byte) (string, error) {
	return Encrypt(derivedKey, dataKey)
}

// DecryptDataKey decrypts the data key with the derived master key
func DecryptDataKey(derivedKey []byte, encryptedDataKey string) ([]byte, error) {
	return Decrypt(derivedKey, encryptedDataKey)
}

// EncryptDSN encrypts a DSN with the data key
func EncryptDSN(dataKey []byte, dsn string) (string, error) {
	return Encrypt(dataKey, []byte(dsn))
}

// DecryptDSN decrypts a DSN with the data key
func DecryptDSN(dataKey []byte, encryptedDSN string) (string, error) {
	plaintext, err := Decrypt(dataKey, encryptedDSN)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// InitializeVault initializes a new vault with a encryption password
// Returns the salt and encrypted data key for storage
func InitializeVault(password string) (salt []byte, encryptedDataKey string, dataKey []byte, err error) {
	salt, err = GenerateSalt()
	if err != nil {
		return nil, "", nil, err
	}

	dataKey, err = GenerateDataKey()
	if err != nil {
		return nil, "", nil, err
	}

	derivedKey := DeriveKey(password, salt)
	encryptedDataKey, err = EncryptDataKey(derivedKey, dataKey)
	if err != nil {
		return nil, "", nil, err
	}

	return salt, encryptedDataKey, dataKey, nil
}

// UnlockVault unlocks a vault with the encryption password
func UnlockVault(password string, salt []byte, encryptedDataKey string) ([]byte, error) {
	derivedKey := DeriveKey(password, salt)
	dataKey, err := DecryptDataKey(derivedKey, encryptedDataKey)
	if err != nil {
		return nil, err
	}
	return dataKey, nil
}
