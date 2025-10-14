package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

type expiryDuration struct {
	Duration *time.Duration
}

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
	d.Duration = &dur
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
	v.Check(input.Expiry.Duration == nil || time.Duration(*input.Expiry.Duration) >= time.Minute*5, "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= accessPrivate, "access", "access type can be one of 0, 1 and 2")
	v.Check(len(input.Scrolls) < 255, "scrolls", "no of scrolls can't be greater than 254")
	for i, scroll := range input.Scrolls {
		v.Check(len(scroll.Content) > 0, fmt.Sprintf("scrolls[%d].content", i), "scroll content can't be empty")
	}
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	jar := &database.ScrollJar{
		Name:   input.Name,
		Access: input.Access,
		Tags:   input.Tags,
	}
	if input.Expiry.Duration != nil {
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(*input.Expiry.Duration),
		}
	}
	err = app.models.ScrollJar.Insert(jar)
	if err != nil {
		panic(err)
	}

	scrollURIs := []string{}
	for i, scroll := range input.Scrolls {
		scroll.ID = int8(i + 1)
		err = app.models.ScrollJar.InsertScroll(jar.ID, &scroll)
		if err != nil {
			panic(err)
		}
		scrollURIs = append(scrollURIs, fmt.Sprintf("https://scrolljar.com/%s/%d", jar.Slug, scroll.ID))
	}

	outputPayload := envelope{
		"uri":     fmt.Sprintf("https://scrolljar.com/%s", jar.Slug),
		"scrolls": scrollURIs,
	}
	app.writeJSON(w, http.StatusOK, outputPayload, nil)
}
