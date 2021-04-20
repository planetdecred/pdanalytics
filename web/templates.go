// Copyright (c) 2018-2019, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package web

import (
	"encoding/hex"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/dcrec"
	"github.com/decred/dcrd/dcrutil/v2"
	"github.com/dustin/go-humanize"
)

const (
	TestnetNetName = "Testnet"
	defaultFolder  = "./views"
)

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

type periodMap struct {
	y          string
	mo         string
	d          string
	h          string
	min        string
	s          string
	sep        string
	pluralizer func(string, int) string
}

var shortPeriods = &periodMap{
	y:   "y",
	mo:  "mo",
	d:   "d",
	h:   "h",
	min: "m",
	s:   "s",
	sep: " ",
	pluralizer: func(s string, count int) string {
		return s
	},
}

// NewTemplates creates a new Templates obj. The default folder is ./views
func NewTemplates(folder string, reload bool, common []string, helpers template.FuncMap) *Templates {
	if folder == "" {
		folder = defaultFolder
	}
	com := make([]string, 0, len(common))
	for _, file := range common {
		// if this is a custom folder and the common file is not found,
		// try and use the one in ./view folder
		fileName := filepath.Join(folder, file+".tmpl")
		if _, err := os.Stat(fileName); os.IsNotExist(err) && folder != defaultFolder {
			if _, err := os.Stat(filepath.Join(defaultFolder, file+".tmpl")); err == nil {
				fileName = filepath.Join(defaultFolder, file+".tmpl")
			}
		}
		com = append(com, fileName)
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

	return &t
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
// writing to the client, otherwise you might as well execute directly into
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
		return TestnetNetName
	}
	return strings.Title(chainParams.Name)
}

func amountAsDecimalPartsTrimmed(v, numPlaces int64, useCommas bool) []string {
	// Filter numPlaces to only allow up to 8 decimal places trimming (eg. 1.12345678)
	if numPlaces > 8 {
		numPlaces = 8
	}

	// Separate values passed in into int.dec parts.
	intpart := v / 1e8
	decpart := v % 1e8

	// Format left side.
	left := strconv.FormatInt(intpart, 10)
	rightWithTail := fmt.Sprintf("%08d", decpart)

	// Reduce precision according to numPlaces.
	if len(rightWithTail) > int(numPlaces) {
		rightWithTail = rightWithTail[0:numPlaces]
	}

	// Separate trailing zeros.
	right := strings.TrimRight(rightWithTail, "0")
	tail := strings.TrimPrefix(rightWithTail, right)

	// Add commas (eg. 3,444.33)
	if useCommas && (len(left) > 3) {
		integerAsInt64, err := strconv.ParseInt(left, 10, 64)
		if err != nil {
			log.Errorf("amountAsDecimalParts comma formatting failed. Input: %v Error: %v", v, err.Error())
			left = "ERROR"
			right = "VALUE"
			tail = ""
			return []string{left, right, tail}
		}
		left = humanize.Comma(integerAsInt64)
	}

	return []string{left, right, tail}
}

var toInt64 = func(v interface{}) int64 {
	switch vt := v.(type) {
	case int64:
		return vt
	case int32:
		return int64(vt)
	case uint32:
		return int64(vt)
	case uint64:
		return int64(vt)
	case int:
		return int64(vt)
	case int16:
		return int64(vt)
	case uint16:
		return int64(vt)
	default:
		return math.MinInt64
	}
}

var longPeriods = &periodMap{
	y:   " year",
	mo:  " month",
	d:   " day",
	h:   " hour",
	min: " minutes",
	s:   " seconds",
	sep: ", ",
	pluralizer: func(s string, count int) string {
		if count == 1 {
			return s
		}
		return s + "s"
	},
}

func formattedDuration(duration time.Duration, str *periodMap) string {
	durationyr := int(duration / (time.Hour * 24 * 365))
	durationmo := int((duration / (time.Hour * 24 * 30)) % 12)
	pl := str.pluralizer
	i := strconv.Itoa
	if durationyr != 0 {
		return i(durationyr) + "y " + i(durationmo) + "mo"
	}

	durationdays := int((duration / time.Hour / 24) % 30)
	if durationmo != 0 {
		return i(durationmo) + pl(str.mo, durationmo) + str.sep + i(durationdays) + pl(str.d, durationdays)
	}

	durationhr := int((duration / time.Hour) % 24)
	if durationdays != 0 {
		return i(durationdays) + pl(str.d, durationdays) + str.sep + i(durationhr) + pl(str.h, durationhr)
	}

	durationmin := int(duration.Minutes()) % 60
	if durationhr != 0 {
		return i(durationhr) + pl(str.h, durationhr) + str.sep + i(durationmin) + pl(str.min, durationmin)
	}

	durationsec := int(duration.Seconds()) % 60
	if (durationhr == 0) && (durationmin != 0) {
		return i(durationmin) + pl(str.min, durationmin) + str.sep + i(durationsec) + pl(str.s, durationsec)
	}
	return i(durationsec) + pl(str.s, durationsec)
}

// MakeTemplateFuncMap defines common template functions that are shered
// accross all the modules. Individual modules can extend this and add
// functions that are specific to the module
func MakeTemplateFuncMap(params *chaincfg.Params) template.FuncMap {
	netTheme := "theme-" + strings.ToLower(netName(params))

	return template.FuncMap{
		"add": func(args ...int64) int64 {
			var sum int64
			for _, a := range args {
				sum += a
			}
			return sum
		},
		"subtract": func(a, b int64) int64 {
			return a - b
		},
		"floatsubtract": func(a, b float64) float64 {
			return a - b
		},
		"intSubtract": func(a, b int) int {
			return a - b
		},
		"divide": func(n, d int64) int64 {
			return n / d
		},
		"divideFloat": func(n, d float64) float64 {
			return n / d
		},
		"multiply": func(a, b int64) int64 {
			return a * b
		},
		"intMultiply": func(a, b int) int {
			return a * b
		},
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
		"redirectToMainnet": func(netName string, message string) bool {
			if netName != "Mainnet" && strings.Contains(message, "mainnet") {
				return true
			}
			return false
		},
		"neq": func (a, b interface{}) bool {
			return a != b
		},
		"redirectToTestnet": func(netName string, message string) bool {
			if netName != TestnetNetName && strings.Contains(message, "testnet") {
				return true
			}
			return false
		},
		"PKAddr2PKHAddr": func(address string) (p2pkh string) {
			// Attempt to decode the pay-to-pubkey address.
			var addr dcrutil.Address
			addr, err := dcrutil.DecodeAddress(address, params)
			if err != nil {
				log.Errorf(err.Error())
				return ""
			}

			// Extract the pubkey hash.
			addrHash := addr.Hash160()

			// Create a new pay-to-pubkey-hash address.
			addrPKH, err := dcrutil.NewAddressPubKeyHash(addrHash[:], params, dcrec.STEcdsaSecp256k1)
			if err != nil {
				log.Errorf(err.Error())
				return ""
			}
			return addrPKH.Address()
		},
		"float64AsDecimalParts": Float64Formatting,
		"amountAsDecimalParts": func(v int64, useCommas bool) []string {
			return Float64Formatting(dcrutil.Amount(v).ToCoin(), 8, useCommas)
		},
		"durationToShortDurationString": func(d time.Duration) string {
			return FormattedDuration(d, shortPeriods)
		},
		"amountAsDecimalPartsTrimmed": amountAsDecimalPartsTrimmed,
		"secondsToLongDurationString": func(d int64) string {
			return formattedDuration(time.Duration(d)*time.Second, longPeriods)
		},
		"secondsToShortDurationString": func(d int64) string {
			return formattedDuration(time.Duration(d)*time.Second, shortPeriods)
		},
		"uint16Mul": func(a uint16, b int) (result int) {
			result = int(a) * b
			return
		},
		"int64": toInt64,
		"intComma": func(v interface{}) string {
			return humanize.Comma(toInt64(v))
		},
		"percentage": func(a, b int64) float64 {
			return (float64(a) / float64(b)) * 100
		},
		"convertByteArrayToString": func(arr []byte) (inString string) {
			inString = hex.EncodeToString(arr)
			return
		},
		"normalizeBalance": func(balance float64) string {
			return fmt.Sprintf("%010.8f DCR", balance)
		},
		"timestamp": func () int64 {
			return time.Now().Unix()
		},
		"removeStartingSlash": func (url string) string {
			if strings.HasPrefix(url, "/") {
				url = url[1:]
			}
			return url
		},
		"toAbsValue": math.Abs,
		"toFloat64": func(x uint32) float64 {
			return float64(x)
		},
		"toInt": func(str string) int {
			intStr, err := strconv.Atoi(str)
			if err != nil {
				return 0
			}
			return intStr
		},
		"x100": func(v float64) float64 {
			return v * 100
		},
		"f32x100": func(v float32) float32 {
			return v * 100
		},
		"TimeConversion": func(a uint64) string {
			if a == 0 {
				return "N/A"
			}
			dateTime := time.Unix(int64(a), 0).UTC()
			return dateTime.Format("2006-01-02 15:04:05 MST")
		},
		"dateTimeWithoutTimeZone": func(a uint64) string {
			if a == 0 {
				return "N/A"
			}
			dateTime := time.Unix(int64(a), 0).UTC()
			return dateTime.Format("2006-01-02 15:04:05")
		},
		"floor": math.Floor,
		"toLowerCase": strings.ToLower,
		"toTitleCase": strings.Title,
	}
}

// Float64Formatting formats a float64 value into multiple strings depending on whether
// boldNumPlaces is provided or not. boldNumPlaces defines the number of decimal
// places to be written with same font as the whole number value of the float.
// If boldNumPlaces is provided the returned slice should have at least four items
// otherwise it should have at least three items. i.e. given v is to 342.12132000,
// numplaces is 8 and boldNumPlaces is set to 2 the following should be returned
// []string{"342", "12", "132", "000"}. If boldNumPlace is not set the returned
// slice should be []string{"342", "12132", "000"}.
func Float64Formatting(v float64, numPlaces int, useCommas bool, boldNumPlaces ...int) []string {
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

func FormattedDuration(duration time.Duration, str *periodMap) string {
	durationyr := int(duration / (time.Hour * 24 * 365))
	durationmo := int((duration / (time.Hour * 24 * 30)) % 12)
	pl := str.pluralizer
	i := strconv.Itoa
	if durationyr != 0 {
		return i(durationyr) + "y " + i(durationmo) + "mo"
	}

	durationdays := int((duration / time.Hour / 24) % 30)
	if durationmo != 0 {
		return i(durationmo) + pl(str.mo, durationmo) + str.sep + i(durationdays) + pl(str.d, durationdays)
	}

	durationhr := int((duration / time.Hour) % 24)
	if durationdays != 0 {
		return i(durationdays) + pl(str.d, durationdays) + str.sep + i(durationhr) + pl(str.h, durationhr)
	}

	durationmin := int(duration.Minutes()) % 60
	if durationhr != 0 {
		return i(durationhr) + pl(str.h, durationhr) + str.sep + i(durationmin) + pl(str.min, durationmin)
	}

	durationsec := int(duration.Seconds()) % 60
	if (durationhr == 0) && (durationmin != 0) {
		return i(durationmin) + pl(str.min, durationmin) + str.sep + i(durationsec) + pl(str.s, durationsec)
	}
	return i(durationsec) + pl(str.s, durationsec)
}
