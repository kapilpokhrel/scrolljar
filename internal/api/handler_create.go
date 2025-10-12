package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type accessType int

func (a accessType) string() (string, bool) {
	switch a {
	case 0:
		return "public", true
	case 1:
		return "unlisted", true
	case 2:
		return "private", true
	default:
		return "", false
	}
}

const (
	accessPublic accessType = iota
	accessUnlisted
	accessPrivate
)

type scroll struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

func (app *Application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string     `json:"name"`
		Access  accessType `json:"access"`
		Expiry  string     `json:"expiry"`
		Scrolls []scroll   `json:"scrolls"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	fmt.Fprintf(w, "%+v\n", input)
}
