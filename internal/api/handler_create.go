package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
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

const (
	accessPublic int = iota
	accessUnlisted
	accessPrivate
)

func (app *Application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string            `json:"name"`
		Access  int               `json:"access"`
		Expiry  expiryDuration    `json:"expiry"`
		Tags    []string          `json:"tags"`
		Scrolls []database.Scroll `json:"scrolls"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	v.Check(input.Expiry >= expiryDuration(time.Minute*5), "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= accessPrivate, "access", "access type can be one of 0, 1 and 2")
	for i, scroll := range input.Scrolls {
		v.Check(len(scroll.Content) > 0, fmt.Sprintf("scrolls[%d].content", i), "scroll content can't be empty")
	}
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	jar := &database.ScrollJar{
		Name:      input.Name,
		Access:    input.Access,
		ExpiresAt: time.Now().Add(time.Duration(input.Expiry)),
		Tags:      input.Tags,
	}

	err = app.models.ScrollJar.Insert(jar)
	if err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
	}
	fmt.Fprintf(w, "%+v\n", input)
}
