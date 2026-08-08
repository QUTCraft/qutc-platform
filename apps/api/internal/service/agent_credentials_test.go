package service

import "testing"

func TestAgentCredentialEncryptionRoundTrip(t *testing.T) {
	key := deriveAgentCredentialKey("test-access-secret")
	plain := "sk-test-provider-key-1234"

	ciphertext, err := encryptAgentCredential(key, plain)
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}
	if ciphertext == plain || ciphertext == "" {
		t.Fatalf("credential was not encrypted: %q", ciphertext)
	}
	decoded, err := decryptAgentCredential(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt credential: %v", err)
	}
	if decoded != plain {
		t.Fatalf("decrypted credential = %q, want %q", decoded, plain)
	}
	if _, err := decryptAgentCredential(deriveAgentCredentialKey("different-secret"), ciphertext); err == nil {
		t.Fatal("credential decrypted with an unrelated secret")
	}
}

func TestValidAgentProviderConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		driver     string
		baseURL    string
		apiKey     string
		model      string
		production bool
		valid      bool
	}{
		{name: "disabled", driver: "disabled", valid: true},
		{name: "development mock", driver: "mock", model: "mock-content-v1", valid: true},
		{name: "production mock", driver: "mock", model: "mock-content-v1", production: true, valid: false},
		{name: "compatible https", driver: "openai_compatible", baseURL: "https://api.example.test/v1", apiKey: "sk-test", model: "example-model", production: true, valid: true},
		{name: "compatible http development", driver: "openai_compatible", baseURL: "http://localhost:8080/v1", apiKey: "sk-test", model: "example-model", valid: true},
		{name: "compatible missing key", driver: "openai_compatible", baseURL: "https://api.example.test/v1", model: "example-model", production: true, valid: false},
		{name: "compatible query rejected", driver: "openai_compatible", baseURL: "https://api.example.test/v1?token=secret", apiKey: "sk-test", model: "example-model", valid: false},
		{name: "production http rejected", driver: "openai_compatible", baseURL: "http://api.example.test/v1", apiKey: "sk-test", model: "example-model", production: true, valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validAgentProviderConfiguration(test.driver, test.baseURL, test.apiKey, test.model, test.production); got != test.valid {
				t.Fatalf("validAgentProviderConfiguration() = %v, want %v", got, test.valid)
			}
		})
	}
}

func TestAgentCredentialHintDoesNotExposeFullCredential(t *testing.T) {
	hint := agentCredentialHint("sk-test-provider-key-1234")
	if hint == "" || hint == "sk-test-provider-key-1234" || hint[len(hint)-4:] != "1234" {
		t.Fatalf("unexpected credential hint: %q", hint)
	}
	if agentCredentialHint("") != "" {
		t.Fatal("empty credential should have an empty hint")
	}
}
