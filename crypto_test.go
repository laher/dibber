package main

import (
	"bytes"
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if len(salt1) != saltLen {
		t.Errorf("salt length = %d, want %d", len(salt1), saltLen)
	}

	// Two salts should be different
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if bytes.Equal(salt1, salt2) {
		t.Error("two salts should not be equal")
	}
}

func TestGenerateDataKey(t *testing.T) {
	key1, err := GenerateDataKey()
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	if len(key1) != dataKeyLen {
		t.Errorf("data key length = %d, want %d", len(key1), dataKeyLen)
	}

	// Two keys should be different
	key2, err := GenerateDataKey()
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	if bytes.Equal(key1, key2) {
		t.Error("two data keys should not be equal")
	}
}

func TestDeriveKey(t *testing.T) {
	password := "test-password-123"
	salt, _ := GenerateSalt()

	key1 := DeriveKey(password, salt)
	if len(key1) != argonKeyLen {
		t.Errorf("derived key length = %d, want %d", len(key1), argonKeyLen)
	}

	// Same password and salt should produce same key
	key2 := DeriveKey(password, salt)
	if !bytes.Equal(key1, key2) {
		t.Error("same password/salt should produce same key")
	}

	// Different password should produce different key
	key3 := DeriveKey("different-password", salt)
	if bytes.Equal(key1, key3) {
		t.Error("different password should produce different key")
	}

	// Different salt should produce different key
	salt2, _ := GenerateSalt()
	key4 := DeriveKey(password, salt2)
	if bytes.Equal(key1, key4) {
		t.Error("different salt should produce different key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("Hello, World! This is a test message.")

	// Encrypt
	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == "" {
		t.Error("ciphertext should not be empty")
	}

	// Decrypt
	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}

	// Wrong key should fail
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 1)
	}
	_, err = Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Error("decrypt with wrong key should fail")
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptNonDeterministic(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("same message")

	// Same plaintext should produce different ciphertexts (due to random nonce)
	ct1, _ := Encrypt(key, plaintext)
	ct2, _ := Encrypt(key, plaintext)

	if ct1 == ct2 {
		t.Error("encryption should be non-deterministic (random nonce)")
	}

	// Both should decrypt to same plaintext
	pt1, _ := Decrypt(key, ct1)
	pt2, _ := Decrypt(key, ct2)
	if !bytes.Equal(pt1, pt2) {
		t.Error("both ciphertexts should decrypt to same plaintext")
	}
}

func TestInitializeVault(t *testing.T) {
	password := "my-secure-password"

	salt, encryptedDataKey, dataKey, err := InitializeVault(password)
	if err != nil {
		t.Fatalf("InitializeVault failed: %v", err)
	}

	if len(salt) != saltLen {
		t.Errorf("salt length = %d, want %d", len(salt), saltLen)
	}
	if encryptedDataKey == "" {
		t.Error("encrypted data key should not be empty")
	}
	if len(dataKey) != dataKeyLen {
		t.Errorf("data key length = %d, want %d", len(dataKey), dataKeyLen)
	}

	// Should be able to unlock with same password
	unlockedKey, err := UnlockVault(password, salt, encryptedDataKey)
	if err != nil {
		t.Fatalf("UnlockVault failed: %v", err)
	}
	if !bytes.Equal(unlockedKey, dataKey) {
		t.Error("unlocked key should match original data key")
	}

	// Wrong password should fail
	_, err = UnlockVault("wrong-password", salt, encryptedDataKey)
	if err == nil {
		t.Error("unlock with wrong password should fail")
	}
}

func TestEncryptDecryptDSN(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	dsn := "user:password@tcp(localhost:3306)/mydb"

	encrypted, err := EncryptDSN(dataKey, dsn)
	if err != nil {
		t.Fatalf("EncryptDSN failed: %v", err)
	}

	decrypted, err := DecryptDSN(dataKey, encrypted)
	if err != nil {
		t.Fatalf("DecryptDSN failed: %v", err)
	}
	if decrypted != dsn {
		t.Errorf("decrypted DSN = %q, want %q", decrypted, dsn)
	}
}

func TestVaultLockClearsKey(t *testing.T) {
	v := NewVault()
	v.dataKey = make([]byte, 32)
	for i := range v.dataKey {
		v.dataKey[i] = byte(i)
	}
	v.connections["test"] = "some-dsn"
	v.isUnlocked = true

	v.Lock()

	if v.isUnlocked {
		t.Error("vault should be locked after Lock()")
	}
	if len(v.connections) != 0 {
		t.Error("connections should be cleared after Lock()")
	}
	// Can't verify key is zeroed perfectly, but at least it should be nil
	if v.dataKey != nil {
		t.Error("dataKey should be nil after Lock()")
	}
}
