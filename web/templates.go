// Copyright (c) 2018-2019, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package web

import (
	"fmt"
	"html/template"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/dustin/go-humanize"
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
		"hashlink": func(hash string, link string) [2]string {
			return [2]string{hash, link}
		},
		"hashStart": func(hash string) string {
			clipLen := 6
			hashLen := len(hash) - clipLen
			if hashLen < 1 {
				return ""
			}
			return hash[0:hashLen]
		},
		"hashEnd": func(hash string) string {
			clipLen := 6
			hashLen := len(hash) - clipLen
			if hashLen < 0 {
				return hash
			}
			return hash[hashLen:]
		},
		"float64AsDecimalParts": float64Formatting,
	}
}

// float64Formatting formats a float64 value into multiple strings depending on whether
// boldNumPlaces is provided or not. boldNumPlaces defines the number of decimal
// places to be written with same font as the whole number value of the float.
// If boldNumPlaces is provided the returned slice should have at least four items
// otherwise it should have at least three items. i.e. given v is to 342.12132000,
// numplaces is 8 and boldNumPlaces is set to 2 the following should be returned
// []string{"342", "12", "132", "000"}. If boldNumPlace is not set the returned
// slice should be []string{"342", "12132", "000"}.
func float64Formatting(v float64, numPlaces int, useCommas bool, boldNumPlaces ...int) []string {
	pow := math.Pow(10, float64(numPlaces))
	formattedVal := math.Round(v*pow) / pow
	clipped := fmt.Sprintf("%."+strconv.Itoa(numPlaces)+"f", formattedVal)
	oldLength := len(clipped)
	clipped = strings.TrimRight(clipped, "0")
	trailingZeros := strings.Repeat("0", oldLength-len(clipped))
	valueChunks := strings.Split(clipped, ".")
	integer := valueChunks[0]

	dec := ""
	if len(valueChunks) > 1 {
		dec = valueChunks[1]
	}

	if useCommas {
		integer = humanize.Comma(int64(formattedVal))
	}

	if len(boldNumPlaces) == 0 {
		return []string{integer, dec, trailingZeros}
	}

	places := boldNumPlaces[0]
	if places > numPlaces {
		return []string{integer, dec, trailingZeros}
	}

	if len(dec) < places {
		places = len(dec)
	}

	return []string{integer, dec[:places], dec[places:], trailingZeros}
}
