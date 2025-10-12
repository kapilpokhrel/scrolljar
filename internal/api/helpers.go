package api

import (
	"encoding/json"
	"maps"
	"net/http"
)

type envelope map[string]any

func (app *Application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	jsonString, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	maps.Copy(w.Header(), headers)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonString)
	w.Write([]byte("\n"))
	return nil
}
