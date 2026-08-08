package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
)

const agentCredentialVersion = "v1"

func deriveAgentCredentialKey(secret string) []byte {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		secret = "development-only-agent-credential-key"
	}
	sum := sha256.Sum256([]byte("qutcraft-platform/agent-provider/" + secret))
	return sum[:]
}

func encryptAgentCredential(key []byte, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create agent credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", fmt.Errorf("create agent credential gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate agent credential nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)
	encoded := append(nonce, ciphertext...)
	return agentCredentialVersion + ":" + base64.RawStdEncoding.EncodeToString(encoded), nil
}

func decryptAgentCredential(key []byte, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	version, encoded, ok := strings.Cut(value, ":")
	if !ok || version != agentCredentialVersion {
		return "", fmt.Errorf("unsupported agent credential format")
	}
	data, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode agent credential: %w", err)
	}
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create agent credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return "", fmt.Errorf("create agent credential gcm: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("agent credential is truncated")
	}
	plaintext, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt agent credential: %w", err)
	}
	return string(plaintext), nil
}

func agentCredentialHint(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= 4 {
		return "••••"
	}
	return "••••••" + string(runes[len(runes)-4:])
}

func validAgentProviderConfiguration(driver, baseURL, apiKey, modelName string, production bool) bool {
	driver = strings.ToLower(strings.TrimSpace(driver))
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	modelName = strings.TrimSpace(modelName)
	if len(baseURL) > 500 || len(apiKey) > 4096 || len(modelName) > 120 {
		return false
	}
	switch driver {
	case "", "disabled":
		return true
	case "mock":
		return !production
	case "openai_compatible":
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return false
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return false
		}
		if production && parsed.Scheme != "https" {
			return false
		}
		return apiKey != "" && modelName != ""
	default:
		return false
	}
}
