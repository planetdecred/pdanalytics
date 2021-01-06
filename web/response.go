package web

import (
	"encoding/json"
	"fmt"
	"net/http"
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
