package portalmanifest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	SchemaV1      = "qutc.portal/v1"
	ThemeSchemaV1 = "qutc.portal-theme/v1"
)

var (
	portalIDPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	semverPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	colorPattern     = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	integrityPattern = regexp.MustCompile(
		`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+={0,2}$`,
	)
)

var allowedCapabilities = map[string]struct{}{
	"organization.read":   {},
	"public_content.read": {},
	"projects.read":       {},
	"assets.read":         {},
	"knowledge.read":      {},
}

type Manifest struct {
	Schema       string   `json:"schema"`
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	DisplayName  string   `json:"display_name"`
	Entry        string   `json:"entry"`
	Theme        ThemeRef `json:"theme"`
	Capabilities []string `json:"capabilities"`
	Fallback     string   `json:"fallback"`
	Integrity    string   `json:"integrity,omitempty"`
}

type ThemeRef struct {
	Mode   string `json:"mode"`
	Tokens string `json:"tokens,omitempty"`
}

type ThemeTokens struct {
	Schema     string          `json:"schema"`
	Color      ThemeColors     `json:"color"`
	Shape      ThemeShape      `json:"shape"`
	Typography ThemeTypography `json:"typography"`
}

type ThemeColors struct {
	Primary   string `json:"primary"`
	OnPrimary string `json:"on_primary"`
	Surface   string `json:"surface"`
	OnSurface string `json:"on_surface"`
}

type ThemeShape struct {
	Small  int `json:"small"`
	Medium int `json:"medium"`
	Large  int `json:"large"`
}

type ThemeTypography struct {
	BodyFamily string `json:"body_family"`
}

type Violation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Parse(data []byte) (Manifest, []Violation) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, []Violation{{Field: "$", Code: "manifest.invalid_json", Message: "Manifest 不是有效 JSON。"}}
	}
	return manifest, Validate(manifest)
}

func Validate(manifest Manifest) []Violation {
	violations := make([]Violation, 0)
	add := func(field, code, message string) {
		violations = append(violations, Violation{Field: field, Code: code, Message: message})
	}

	if manifest.Schema != SchemaV1 {
		add("schema", "manifest.schema_unsupported", "schema 必须为 "+SchemaV1+"。")
	}
	if !portalIDPattern.MatchString(manifest.ID) || len(manifest.ID) > 64 {
		add("id", "manifest.id_invalid", "id 仅允许小写字母、数字和单个连字符，最长 64 字符。")
	}
	if !semverPattern.MatchString(manifest.Version) || len(manifest.Version) > 64 {
		add("version", "manifest.version_invalid", "version 必须是有效 SemVer。")
	}
	if count := utf8.RuneCountInString(strings.TrimSpace(manifest.DisplayName)); count < 1 || count > 80 {
		add("display_name", "manifest.display_name_invalid", "display_name 必须为 1 到 80 个字符。")
	}
	if reason := validateStaticPath(manifest.Entry, ".html"); reason != "" {
		add("entry", "manifest.entry_invalid", reason)
	}
	if manifest.Theme.Mode != "md3" && manifest.Theme.Mode != "custom" {
		add("theme.mode", "manifest.theme_mode_invalid", "theme.mode 仅支持 md3 或 custom。")
	}
	if manifest.Theme.Mode == "custom" {
		if reason := validateStaticPath(manifest.Theme.Tokens, ".json"); reason != "" {
			add("theme.tokens", "manifest.theme_tokens_invalid", reason)
		}
	} else if manifest.Theme.Tokens != "" {
		add("theme.tokens", "manifest.theme_tokens_not_allowed", "md3 模式不能声明自定义 Token。")
	}
	if len(manifest.Capabilities) == 0 || len(manifest.Capabilities) > len(allowedCapabilities) {
		add("capabilities", "manifest.capabilities_invalid", "capabilities 必须包含 1 到 6 个公开能力。")
	}
	seen := map[string]struct{}{}
	for index, capability := range manifest.Capabilities {
		field := fmt.Sprintf("capabilities[%d]", index)
		if _, allowed := allowedCapabilities[capability]; !allowed {
			add(field, "manifest.capability_not_allowed", "能力不在 Portal v1 公开白名单中。")
		}
		if _, duplicate := seen[capability]; duplicate {
			add(field, "manifest.capability_duplicate", "能力不能重复声明。")
		}
		seen[capability] = struct{}{}
	}
	if manifest.Fallback != "md3" {
		add("fallback", "manifest.fallback_invalid", "fallback 必须为 md3。")
	}
	if manifest.Integrity != "" && !integrityPattern.MatchString(manifest.Integrity) {
		add("integrity", "manifest.integrity_invalid", "integrity 必须使用 sha256、sha384 或 sha512 SRI 格式。")
	}
	return violations
}

func ParseTheme(data []byte) (ThemeTokens, []Violation) {
	var tokens ThemeTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return ThemeTokens{}, []Violation{{Field: "$", Code: "theme.invalid_json", Message: "主题 Token 不是有效 JSON。"}}
	}
	return tokens, ValidateTheme(tokens)
}

func ValidateTheme(tokens ThemeTokens) []Violation {
	violations := make([]Violation, 0)
	add := func(field, code, message string) {
		violations = append(violations, Violation{Field: field, Code: code, Message: message})
	}
	if tokens.Schema != ThemeSchemaV1 {
		add("schema", "theme.schema_unsupported", "schema 必须为 "+ThemeSchemaV1+"。")
	}
	for field, value := range map[string]string{
		"color.primary": tokens.Color.Primary, "color.on_primary": tokens.Color.OnPrimary,
		"color.surface": tokens.Color.Surface, "color.on_surface": tokens.Color.OnSurface,
	} {
		if !colorPattern.MatchString(value) {
			add(field, "theme.color_invalid", "颜色必须使用 #RRGGBB 格式。")
		}
	}
	if tokens.Shape.Small < 0 || tokens.Shape.Medium < tokens.Shape.Small || tokens.Shape.Large < tokens.Shape.Medium || tokens.Shape.Large > 48 {
		add("shape", "theme.shape_invalid", "圆角必须满足 0 ≤ small ≤ medium ≤ large ≤ 48。")
	}
	family := strings.TrimSpace(tokens.Typography.BodyFamily)
	lowerFamily := strings.ToLower(family)
	if family == "" || utf8.RuneCountInString(family) > 200 || strings.ContainsAny(family, "{};") || strings.Contains(lowerFamily, "url(") {
		add("typography.body_family", "theme.font_family_invalid", "字体族不能为空、不得超过 200 字符，也不能包含 URL 或 CSS 声明。")
	}
	return violations
}

func validateStaticPath(value, extension string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "路径必须是以单个 / 开头的同源绝对路径。"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "路径不能包含外部来源、查询参数或片段。"
	}
	unescaped, err := url.PathUnescape(parsed.Path)
	if err != nil || strings.Contains(unescaped, "\x00") || path.Clean(unescaped) != unescaped {
		return "路径包含无效编码或目录穿越片段。"
	}
	if !strings.EqualFold(path.Ext(unescaped), extension) {
		return "路径必须指向 " + extension + " 静态资源。"
	}
	return ""
}
