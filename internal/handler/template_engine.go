package handler

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TemplateEngine handles parsing and rendering of HTML templates.
type TemplateEngine struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewTemplateEngine creates a new template engine and loads all templates.
func NewTemplateEngine(templatesDir string, logger *zap.Logger) (*TemplateEngine, error) {
	te := &TemplateEngine{
		templates: make(map[string]*template.Template),
		logger:    logger,
	}

	// Define template functions
	te.funcMap = template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"gt": func(a, b float64) bool {
			return a > b
		},
		"lt": func(a, b float64) bool {
			return a < b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"printf": fmt.Sprintf,
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"derefInt": func(i *int) int {
			if i == nil {
				return 0
			}
			return *i
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			if maxLen <= 3 {
				return s[:maxLen]
			}
			return s[:maxLen-3] + "..."
		},
		"humanize": func(s string) string {
			clean := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", " "), "-", " ")
			parts := strings.Fields(clean)
			for i := range parts {
				part := parts[i]
				if len(part) == 0 {
					continue
				}
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
			return strings.Join(parts, " ")
		},
	}

	if err := te.loadTemplates(templatesDir); err != nil {
		return nil, err
	}

	return te, nil
}

// loadTemplates loads all template files from the templates directory.
func (te *TemplateEngine) loadTemplates(templatesDir string) error {
	// Load base layout
	baseLayout := filepath.Join(templatesDir, "layouts", "base.html")

	// Load components
	componentsPattern := filepath.Join(templatesDir, "components", "*.html")
	componentFiles, err := filepath.Glob(componentsPattern)
	if err != nil {
		return fmt.Errorf("failed to glob components: %w", err)
	}

	// Load each page template with the base layout and components
	pagesDir := filepath.Join(templatesDir, "pages")
	err = filepath.WalkDir(pagesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Get template name from filename (e.g., "login.html" -> "login")
		name := strings.TrimSuffix(filepath.Base(path), ".html")

		// Combine: base layout + components + page template
		files := []string{baseLayout}
		files = append(files, componentFiles...)
		files = append(files, path)

		tmpl, err := template.New(filepath.Base(baseLayout)).Funcs(te.funcMap).ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		te.mu.Lock()
		te.templates[name] = tmpl
		te.mu.Unlock()

		te.logger.Debug("loaded template", zap.String("name", name))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load page templates: %w", err)
	}

	te.logger.Info("templates loaded", zap.Int("count", len(te.templates)))
	return nil
}

// Render renders a template by name with the given data.
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	te.mu.RLock()
	tmpl, ok := te.templates[name]
	te.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	// Execute the "base" template which includes the page content
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return nil
}

// HasTemplate checks if a template exists.
func (te *TemplateEngine) HasTemplate(name string) bool {
	te.mu.RLock()
	defer te.mu.RUnlock()
	_, ok := te.templates[name]
	return ok
}
