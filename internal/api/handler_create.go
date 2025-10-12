package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/validator"
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

type expiryDuration time.Duration

var errInvalidExpiryFormat = errors.New("invalid expiry duration foramt")

func (d *expiryDuration) UnmarshalJSON(jsonValue []byte) error {
	val, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return errInvalidExpiryFormat
	}
	dur, err := time.ParseDuration(val)
	if err != nil {
		return errInvalidExpiryFormat
	}
	*d = expiryDuration(dur)
	return nil
}

type scroll struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

func (app *Application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name       string         `json:"name"`
		Access     accessType     `json:"access"`
		Expiry     expiryDuration `json:"expiry"`
		CustomSlug string         `json:"custom_slug"`
		Tags       []string       `json:"tags"`
		Scrolls    []scroll       `json:"scrolls"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	}

	v := validator.New()
	v.Check(input.Expiry >= expiryDuration(time.Minute*5), "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= accessPrivate, "access", "access type can be one of 0, 1 and 2")
	for i, scroll := range input.Scrolls {
		v.Check(len(scroll.Content) > 0, fmt.Sprintf("scrolls[%d].content", i), "scroll content can't be empty")
	}
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
	}

	fmt.Fprintf(w, "%+v\n", input)
}
