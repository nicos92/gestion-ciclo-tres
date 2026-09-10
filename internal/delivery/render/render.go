package render

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

type Renderer struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

func New(fsys embed.FS) (*Renderer, error) {
	funcMap := template.FuncMap{
		"formatDate": func(t interface{}) string {
			if t == nil {
				return ""
			}
			switch v := t.(type) {
			case string:
				if len(v) >= 10 {
					return v[:10]
				}
				return v
			default:
				return fmt.Sprintf("%v", t)
			}
		},
		"formatMoney": func(v interface{}) string {
			switch n := v.(type) {
			case float64:
				return formatFloat(n)
			case int:
				return fmt.Sprintf("%d", n)
			case int64:
				return fmt.Sprintf("%d", n)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		"nl2br": func(s string) template.HTML {
			s = template.HTMLEscapeString(s)
			s = strings.ReplaceAll(s, "\n", "<br>")
			return template.HTML(s)
		},
		"toFloat": func(v interface{}) float64 {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			default:
				return 0
			}
		},
	}

	templatesDir, err := fs.Sub(fsys, "web/templates")
	if err != nil {
		return nil, fmt.Errorf("templates sub fs: %w", err)
	}

	templates := make(map[string]*template.Template)

	layoutContent, err := fs.ReadFile(templatesDir, "layout.html")
	if err != nil {
		return nil, fmt.Errorf("read layout.html: %w", err)
	}

	entries, err := fs.ReadDir(templatesDir, ".")
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "layout.html" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".html")

		pageContent, err := fs.ReadFile(templatesDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		// Also read partials
		partialsDir, _ := fs.Sub(templatesDir, "partials")
		var partialContents []string
		if partialsDir != nil {
			partialEntries, _ := fs.ReadDir(partialsDir, ".")
			for _, pe := range partialEntries {
				if pe.IsDir() {
					continue
				}
				pc, err := fs.ReadFile(partialsDir, pe.Name())
				if err == nil {
					partialContents = append(partialContents, string(pc))
				}
			}
		}

		combined := string(layoutContent) + "\n" + string(pageContent)
		for _, pc := range partialContents {
			combined += "\n" + pc
		}

		t, err := template.New(name).Funcs(funcMap).Parse(combined)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		templates[name] = t
	}

	return &Renderer{templates: templates, funcMap: funcMap}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any, statusCode int) {
	t, ok := r.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func formatFloat(f float64) string {
	whole := int64(f)
	decimal := int64((f - float64(whole)) * 100)
	if decimal < 0 {
		decimal = -decimal
	}
	return fmt.Sprintf("%d.%02d", whole, decimal)
}
