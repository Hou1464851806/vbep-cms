package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sync"
)

type TemplateEngine struct {
	mu          sync.RWMutex
	templates   *template.Template
	templateDir string
}

func NewTemplateEngine(dir string) (*TemplateEngine, error) {
	engine := &TemplateEngine{templateDir: dir}
	err := engine.Reload()
	if err != nil {
		return nil, fmt.Errorf("failed to reload templates: %v", err)
	}
	return engine, nil
}

// Reload 提供热更新机制，可在其他地方调用来更新模板
func (e *TemplateEngine) Reload() error {
	pattern := filepath.Join(e.templateDir, "*.html")
	newTemplates, err := template.ParseGlob(pattern)
	if err != nil {
		return fmt.Errorf("failed to parse html files: %v", err)
	}

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

	t := e.templates.Lookup(templateName)
	if t == nil {
		return "", fmt.Errorf("failed to find template of [%s]", templateName)
	}

	clone, err := e.templates.Clone()
	if err != nil {
		return "", fmt.Errorf("failed to clone templates: %v", err)
	}

	_, err = clone.AddParseTree("content", t.Tree)
	if err != nil {
		return "", fmt.Errorf("failed to add content to base template: %v", err)
	}

	var buf bytes.Buffer

	err = clone.ExecuteTemplate(&buf, "layout", data)
	if err != nil {
		return "", fmt.Errorf("failed to execute final html: %v", err)
	}

	return buf.String(), nil
}
