package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"server/logger"
	"sync"
)

type TemplateEngine struct {
	mu          sync.RWMutex
	templates   map[string]*template.Template
	templateDir string
}

func NewTemplateEngine(dir string) *TemplateEngine {
	engine := &TemplateEngine{templateDir: dir}
	err := engine.Reload()
	if err != nil {
		logger.Errorf("Failed to reload templates: %v", err)
	}
	return engine
}

// Reload 提供热更新机制，可在其他地方调用来更新模板
func (e *TemplateEngine) Reload() error {
	layoutPath := filepath.Join(e.templateDir, "layout.html")

	pattern := filepath.Join(e.templateDir, "*.html")
	templateFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob template files: %v", err)
	}

	newTemplates := make(map[string]*template.Template)
	for _, file := range templateFiles {
		templateName := filepath.Base(file)
		if templateName == "layout.html" {
			continue
		}

		tmpl, err := template.ParseFiles(layoutPath, file)
		if err != nil {
			return fmt.Errorf("failed to parse html template %s: %v", templateName, err)
		}

		newTemplates[templateName] = tmpl
	}

	logger.Debugf("All templates: %v", newTemplates)

	// 重载模板时加写锁，是为了保证模板加载的操作唯一性
	e.mu.Lock()
	defer e.mu.Unlock()
	e.templates = newTemplates

	return nil
}

// Render 使用 data 的内容渲染 templateName 的模板
func (e *TemplateEngine) Render(templateName string, data Data) (string, error) {
	// 渲染模板时加读锁，是为了防止模板被中途修改，但不阻塞其余并发的模板渲染
	e.mu.RLock()
	defer e.mu.RUnlock()

	t, exists := e.templates[templateName]
	if !exists {
		return "", fmt.Errorf("failed to find template [%s]", templateName)
	}

	var buf bytes.Buffer
	err := t.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		return "", fmt.Errorf("failed to execute final html: %v", err)
	}

	return buf.String(), nil
}
