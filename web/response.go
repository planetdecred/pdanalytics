package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

func RenderErrorfJSON(res http.ResponseWriter, errorMessage string, args ...interface{}) {
	data := map[string]interface{}{
		"error": fmt.Sprintf(errorMessage, args...),
	}

	RenderJSON(res, data)
}

func RenderJSON(res http.ResponseWriter, data interface{}) {
	d, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error marshalling data: %s", err.Error())
	}

	res.Header().Set("Content-Type", "application/json")
	_, _ = res.Write(d)
}

// RenderJSONBytes prepares the headers for pre-encoded JSON and writes the JSON
// bytes.
func RenderJSONBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write(data)
	if err != nil {
		// Filter out broken pipe (user pressed "stop") errors
		if _, ok := err.(*net.OpError); ok {
			if strings.Contains(err.Error(), "broken pipe") {
				return
			}
		}
		log.Warnf("ResponseWriter.Write error: %v", err)
	}
}
