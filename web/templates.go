// Copyright (c) 2018-2019, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package web

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
)

const testnetNetName = "Testnet"

type pageTemplate struct {
	file     string
	template *template.Template
}

type Templates struct {
	templates map[string]pageTemplate
	common    []string
	folder    string
	helpers   template.FuncMap
	Exec      func(string, interface{}) (string, error)
}

func newTemplates(folder string, reload bool, common []string, helpers template.FuncMap) Templates {
	com := make([]string, 0, len(common))
	for _, file := range common {
		com = append(com, filepath.Join(folder, file+".tmpl"))
	}
	t := Templates{
		templates: make(map[string]pageTemplate),
		common:    com,
		folder:    folder,
		helpers:   helpers,
	}
	t.Exec = t.ExecTemplateToString
	if reload {
		t.Exec = t.ExecWithReload
	}

	return t
}

func (t *Templates) AddTemplate(name string) error {
	fileName := filepath.Join(t.folder, name+".tmpl")
	files := append(t.common, fileName)
	temp, err := template.New(name).Funcs(t.helpers).ParseFiles(files...)
	if err == nil {
		t.templates[name] = pageTemplate{
			file:     fileName,
			template: temp,
		}
	}
	return err
}

func (t *Templates) ReloadTemplates() error {
	var errorStrings []string
	for fileName := range t.templates {
		err := t.AddTemplate(fileName)
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
		}
	}
	if errorStrings == nil {
		return nil
	}
	return fmt.Errorf(strings.Join(errorStrings, " | "))
}

// ExecTemplateToString executes the associated input template using the
// supplied data, and writes the result into a string. If the template fails to
// execute or isn't found, a non-nil error will be returned. Check it before
// writing to theclient, otherwise you might as well execute directly into
// your response writer instead of the internal buffer of this function.
func (t *Templates) ExecTemplateToString(name string, data interface{}) (string, error) {
	temp, ok := t.templates[name]
	if !ok {
		return "", fmt.Errorf("Template %s not known", name)
	}
	var page strings.Builder
	err := temp.template.ExecuteTemplate(&page, name, data)
	return page.String(), err
}

// ExecWithReload is the same as execTemplateToString, but will reload the
// template first.
func (t *Templates) ExecWithReload(name string, data interface{}) (string, error) {
	err := t.AddTemplate(name)
	if err != nil {
		return "", fmt.Errorf("execWithReload: %v", err)
	}
	log.Debugf("reloaded HTML template %q", name)
	return t.ExecTemplateToString(name, data)
}

// netName returns the name used when referring to a decred network.
func netName(chainParams *chaincfg.Params) string {
	if chainParams == nil {
		return "invalid"
	}
	if strings.HasPrefix(strings.ToLower(chainParams.Name), "testnet") {
		return testnetNetName
	}
	return strings.Title(chainParams.Name)
}

func makeTemplateFuncMap(params *chaincfg.Params) template.FuncMap {
	netTheme := "theme-" + strings.ToLower(netName(params))

	return template.FuncMap{
		"theme": func() string {
			return netTheme
		},
	}
}
