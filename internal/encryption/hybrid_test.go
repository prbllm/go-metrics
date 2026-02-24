package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateRSAKeyPairFiles(t *testing.T) (pubPath, privPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key")

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err, "failed to marshal public key")
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	tempDir := t.TempDir()

	privPath = tempDir + "/private.pem"
	pubPath = tempDir + "/public.pem"

	require.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(privBlock), 0600))
	require.NoError(t, os.WriteFile(pubPath, pem.EncodeToMemory(pubBlock), 0644))

	return pubPath, privPath
}

func TestEncryptDecryptHybrid_Roundtrip(t *testing.T) {
	pubPath, privPath := generateRSAKeyPairFiles(t)

	plaintext := []byte("test hybrid encryption payload")

	encKey, ciphertext, err := EncryptHybrid(pubPath, plaintext)
	require.NoError(t, err, "EncryptHybrid should succeed")
	require.NotEmpty(t, encKey, "encrypted key should not be empty")
	require.NotEmpty(t, ciphertext, "ciphertext should not be empty")

	decrypted, err := DecryptHybrid(privPath, encKey, ciphertext)
	require.NoError(t, err, "DecryptHybrid should succeed")
	require.Equal(t, plaintext, decrypted, "decrypted data should match original plaintext")
}

func TestEncryptHybrid_InvalidPublicKeyPath(t *testing.T) {
	plaintext := []byte("data")

	encKey, ciphertext, err := EncryptHybrid("non-existent-public.pem", plaintext)
	require.Error(t, err, "EncryptHybrid should fail for invalid public key path")
	require.Nil(t, encKey, "encrypted key should be nil on error")
	require.Nil(t, ciphertext, "ciphertext should be nil on error")
}

func TestDecryptHybrid_InvalidPrivateKeyPath(t *testing.T) {
	encKey := []byte("some-encrypted-key")
	ciphertext := []byte("some-ciphertext")

	plaintext, err := DecryptHybrid("non-existent-private.pem", encKey, ciphertext)
	require.Error(t, err, "DecryptHybrid should fail for invalid private key path")
	require.Nil(t, plaintext, "plaintext should be nil on error")
}

func TestDecryptHybrid_InvalidPrivateKeyPEM(t *testing.T) {
	tempDir := t.TempDir()
	badPrivPath := tempDir + "/bad-private.pem"

	require.NoError(t, os.WriteFile(badPrivPath, []byte("not a PEM key"), 0600))

	encKey := []byte("some-encrypted-key")
	ciphertext := []byte("some-ciphertext")

	plaintext, err := DecryptHybrid(badPrivPath, encKey, ciphertext)
	require.Error(t, err, "DecryptHybrid should fail for invalid private key PEM")
	require.Nil(t, plaintext, "plaintext should be nil on error")
}

func TestDecryptHybrid_InvalidCombinedLength(t *testing.T) {
	_, privPath := generateRSAKeyPairFiles(t)

	privKey, err := loadPrivateKey(privPath)
	require.NoError(t, err, "failed to load private key")

	combined := make([]byte, 16)
	_, err = rand.Read(combined)
	require.NoError(t, err, "failed to fill combined bytes")

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &privKey.PublicKey, combined, nil)
	require.NoError(t, err, "EncryptOAEP should succeed on short combined buffer")

	ciphertext := []byte("does-not-matter")

	plaintext, err := DecryptHybrid(privPath, encKey, ciphertext)
	require.Error(t, err, "DecryptHybrid should fail for invalid combined key/nonce length")
	require.Nil(t, plaintext, "plaintext should be nil on error")
}
