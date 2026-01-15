package sanitize

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
)

// SanitizationOptions configura qué datos sanitizar
type SanitizationOptions struct {
	MaskURLTokens      bool     // Oculta tokens en URLs
	FilterEnvVars      []string // Variables de entorno a filtrar
	RedactWindowTitles bool     // Oculta títulos sensibles
	MaskPaths          bool     // Oculta rutas de archivos personales
}

// DefaultOptions retorna configuración segura por defecto
func DefaultOptions() SanitizationOptions {
	return SanitizationOptions{
		MaskURLTokens: true,
		FilterEnvVars: []string{
			"API_KEY", "APIKEY", "SECRET", "PASSWORD", "PASSWD",
			"TOKEN", "AUTH", "CREDENTIALS", "AWS_SECRET_ACCESS_KEY",
			"GITHUB_TOKEN", "SLACK_TOKEN", "OPENAI_API_KEY",
		},
		RedactWindowTitles: false, // Default false to keep usability unless requested
		MaskPaths:          true,
	}
}

// Sanitizer maneja la sanitización de snapshots
type Sanitizer struct {
	opts SanitizationOptions
}

// NewSanitizer crea un nuevo sanitizador
func NewSanitizer(opts SanitizationOptions) *Sanitizer {
	return &Sanitizer{opts: opts}
}

// SanitizeSnapshot sanitiza un snapshot completo
func (s *Sanitizer) SanitizeSnapshot(snap *core.Snapshot) {
	if s.opts.MaskURLTokens {
		s.sanitizeBrowserTabs(snap.BrowserTabs)
	}

	if len(s.opts.FilterEnvVars) > 0 {
		s.sanitizeTerminals(snap.Terminals)
	}

	if s.opts.RedactWindowTitles {
		s.sanitizeWindows(snap.Windows)
	}

	if s.opts.MaskPaths {
		s.sanitizePaths(snap)
	}
}

// sanitizeBrowserTabs oculta tokens en URLs
func (s *Sanitizer) sanitizeBrowserTabs(tabs []core.BrowserTab) {
	for i := range tabs {
		tabs[i].URL = s.maskSensitiveURL(tabs[i].URL)
	}
}

// maskSensitiveURL oculta parámetros sensibles en URLs
func (s *Sanitizer) maskSensitiveURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Si no se puede parsear, al menos intentar regex básico
		return s.maskURLRegex(rawURL)
	}

	// Sanitizar query parameters
	query := parsed.Query()
	sensitiveParams := []string{
		"token", "key", "secret", "apikey", "api_key",
		"access_token", "auth", "password", "passwd",
		"credentials", "session", "jwt",
	}

	for _, param := range sensitiveParams {
		if query.Has(param) {
			query.Set(param, "***REDACTED***")
		}
	}

	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// maskURLRegex usa regex como fallback
func (s *Sanitizer) maskURLRegex(rawURL string) string {
	// Pattern para detectar parámetros sensibles
	re := regexp.MustCompile(`([?&](token|key|secret|apikey|api_key|access_token|auth|password|passwd|session|jwt)=)[^&\s]+`)
	return re.ReplaceAllString(rawURL, "${1}***REDACTED***")
}

// sanitizeTerminals filtra variables de entorno sensibles
func (s *Sanitizer) sanitizeTerminals(terminals []core.Terminal) {
	for i := range terminals {
		if terminals[i].EnvVars == nil {
			continue
		}

		for _, sensitiveKey := range s.opts.FilterEnvVars {
			for key := range terminals[i].EnvVars {
				// Case-insensitive matching
				if strings.EqualFold(key, sensitiveKey) {
					terminals[i].EnvVars[key] = "***REDACTED***"
				}
				// También busca keys que contengan las palabras sensibles
				if containsInsensitive(key, sensitiveKey) {
					terminals[i].EnvVars[key] = "***REDACTED***"
				}
			}
		}
	}
}

// sanitizeWindows oculta información sensible en títulos
func (s *Sanitizer) sanitizeWindows(windows []core.Window) {
	for i := range windows {
		windows[i].WindowTitle = s.maskSensitiveTitle(windows[i].WindowTitle)
	}
}

// maskSensitiveTitle detecta y oculta información sensible en títulos
func (s *Sanitizer) maskSensitiveTitle(title string) string {
	// Patrones comunes de información sensible en títulos
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Emails
		{regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), "***EMAIL***"},
		// IPs
		{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "***IP***"},
		// Tokens que parecen hexadecimales largos
		{regexp.MustCompile(`\b[a-fA-F0-9]{32,}\b`), "***TOKEN***"},
	}

	result := title
	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.replacement)
	}
	return result
}

// sanitizePaths oculta rutas de usuario
func (s *Sanitizer) sanitizePaths(snap *core.Snapshot) {
	// Detectar username común en rutas
	userPattern := regexp.MustCompile(`(?i)(C:\\Users\\|/home/|/Users/)([^\\\/]+)`)

	// Sanitizar rutas en ventanas
	for i := range snap.Windows {
		snap.Windows[i].AppPath = userPattern.ReplaceAllString(
			snap.Windows[i].AppPath,
			"${1}***USER***",
		)
	}

	// Sanitizar rutas en terminales
	for i := range snap.Terminals {
		snap.Terminals[i].WorkingDirectory = userPattern.ReplaceAllString(
			snap.Terminals[i].WorkingDirectory,
			"${1}***USER***",
		)
	}

	// Sanitizar rutas en IDE files
	for i := range snap.IDEFiles {
		snap.IDEFiles[i].FilePath = userPattern.ReplaceAllString(
			snap.IDEFiles[i].FilePath,
			"${1}***USER***",
		)
	}

	// Sanitizar git repo path
	snap.GitRepo = userPattern.ReplaceAllString(snap.GitRepo, "${1}***USER***")
}

// containsInsensitive verifica si s contiene substr (case-insensitive)
func containsInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
