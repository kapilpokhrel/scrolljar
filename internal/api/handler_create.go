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
	accessPrivate
)

func (app *Application) postCreateScrollJarHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string            `json:"name"`
		Access   int               `json:"access"`
		Password string            `json:"password"`
		Expiry   expiryDuration    `json:"expiry"`
		Tags     []string          `json:"tags"`
		Scrolls  []database.Scroll `json:"scrolls"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	v.Check(input.Expiry.Duration == nil || time.Duration(*input.Expiry.Duration) >= time.Minute*5, "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= accessPrivate, "access", "access type can be one of 0, 1")
	v.Check(input.Access == accessPublic || len(input.Password) != 0, "password", "password can't be empty when access is private")
	v.Check(len(input.Scrolls) < 255, "scrolls", "no of scrolls can't be greater than 254")
	for i, scroll := range input.Scrolls {
		v.Check(len(scroll.Content) > 0, fmt.Sprintf("scrolls[%d].content", i), "scroll content can't be empty")
	}
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	pwHash, err := hashPassword(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	jar := &database.ScrollJar{
		Name:         input.Name,
		Access:       input.Access,
		PasswordHash: pwHash,
		Tags:         input.Tags,
	}
	user := app.contextGetUser(r)
	if user != nil {
		userID := user.ID
		jar.UserID = &userID
	}

	if input.Expiry.Duration != nil {
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(*input.Expiry.Duration),
		}
	}
	err = app.models.ScrollJar.Insert(jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.getJarURI(jar)
	scrollURIs := []string{}
	for _, scroll := range input.Scrolls {
		scroll.Jar = jar
		err = app.models.ScrollJar.InsertScroll(&scroll)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		app.getScrollURI(&scroll)
		scrollURIs = append(scrollURIs, scroll.URI)
	}

	outputPayload := envelope{
		"uri":     jar.URI,
		"scrolls": scrollURIs,
	}
	app.writeJSON(w, http.StatusOK, outputPayload, nil)
}

func (app *Application) postCreateScrollHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Format  string `json:"format"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	id := app.readIDParam(r)

	jar := database.ScrollJar{
		ID: id,
	}
	err = app.models.ScrollJar.Get(&jar)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	v := validator.New()
	v.Check(len(input.Content) > 0, "content", "content can't be empty")
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	scroll := database.Scroll{
		Title:   input.Title,
		Content: input.Content,
		Format:  input.Format,
		Jar:     &jar,
	}

	err = app.models.ScrollJar.InsertScroll(&scroll)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	app.getScrollURI(&scroll)

	env := envelope{"scroll": scroll}

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
