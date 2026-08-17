package portalmanifest

import "testing"

func validManifest() Manifest {
	return Manifest{
		Schema:      SchemaV1,
		ID:          "campus-club",
		Version:     "0.1.0",
		DisplayName: "Campus Club Portal",
		Entry:       "/portals/campus-club/index.html",
		Theme:       ThemeRef{Mode: "custom", Tokens: "/portals/campus-club/theme.json"},
		Capabilities: []string{
			"organization.read", "public_content.read", "projects.read",
			"assets.read", "knowledge.read",
		},
		Fallback: "md3",
	}
}

func TestValidateManifestV1(t *testing.T) {
	if violations := Validate(validManifest()); len(violations) != 0 {
		t.Fatalf("valid manifest violations = %+v", violations)
	}
}

func TestValidateManifestRejectsUnsafeDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		code   string
	}{
		{name: "external entry", mutate: func(value *Manifest) { value.Entry = "https://evil.invalid/index.html" }, code: "manifest.entry_invalid"},
		{name: "protocol relative entry", mutate: func(value *Manifest) { value.Entry = "//evil.invalid/index.html" }, code: "manifest.entry_invalid"},
		{name: "encoded traversal", mutate: func(value *Manifest) { value.Entry = "/portals/%2e%2e/admin/index.html" }, code: "manifest.entry_invalid"},
		{name: "admin capability", mutate: func(value *Manifest) { value.Capabilities = []string{"admin.read"} }, code: "manifest.capability_not_allowed"},
		{name: "server command", mutate: func(value *Manifest) { value.Capabilities = []string{"server.command"} }, code: "manifest.capability_not_allowed"},
		{name: "duplicate capability", mutate: func(value *Manifest) { value.Capabilities = []string{"organization.read", "organization.read"} }, code: "manifest.capability_duplicate"},
		{name: "custom theme without tokens", mutate: func(value *Manifest) { value.Theme.Tokens = "" }, code: "manifest.theme_tokens_invalid"},
		{name: "md3 with custom tokens", mutate: func(value *Manifest) { value.Theme.Mode = "md3" }, code: "manifest.theme_tokens_not_allowed"},
		{name: "invalid fallback", mutate: func(value *Manifest) { value.Fallback = "custom" }, code: "manifest.fallback_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validManifest()
			test.mutate(&manifest)
			if !containsViolation(Validate(manifest), test.code) {
				t.Fatalf("violations did not contain %q", test.code)
			}
		})
	}
}

func TestValidateThemeTokens(t *testing.T) {
	valid := ThemeTokens{
		Schema:     ThemeSchemaV1,
		Color:      ThemeColors{Primary: "#2d7d1a", OnPrimary: "#ffffff", Surface: "#f3f7ee", OnSurface: "#172011"},
		Shape:      ThemeShape{Small: 8, Medium: 12, Large: 20},
		Typography: ThemeTypography{BodyFamily: "Noto Sans SC, system-ui, sans-serif"},
	}
	if violations := ValidateTheme(valid); len(violations) != 0 {
		t.Fatalf("valid theme violations = %+v", violations)
	}
	valid.Typography.BodyFamily = "url(https://evil.invalid/font.woff2)"
	if !containsViolation(ValidateTheme(valid), "theme.font_family_invalid") {
		t.Fatal("remote font URL was not rejected")
	}
}

func containsViolation(violations []Violation, code string) bool {
	for _, violation := range violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
