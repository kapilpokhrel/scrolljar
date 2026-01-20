package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) CreateJar(w http.ResponseWriter, r *http.Request) {
	input := spec.JarCreationType{}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(input.Expiry.Duration == nil || time.Duration(*input.Expiry.Duration) >= time.Minute*5, "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= spec.AccessPrivate, "access", "access type can be one of 0, 1")
	v.Check(input.Access == spec.AccessPublic || len(input.Password) != 0, "password", "password can't be empty when access is private")
	v.Check(len(input.Scrolls) < 255, "scrolls", "no of scrolls can't be greater than 254")
	for i, scroll := range input.Scrolls {
		v.Check(len(scroll.Content) > 0, fmt.Sprintf("scrolls[%d].content", i), "scroll content can't be empty")
	}

	user := app.contextGetUser(r)

	DurYear := time.Hour * 25 * 365
	v.Check(user != nil || input.Expiry.Duration == nil || *(input.Expiry.Duration) < DurYear, "expiry", "Duration of anonymouns jar must be less than a yaer")
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	jar := &database.ScrollJar{}
	jar.Name = input.Name
	jar.Access = input.Access
	jar.Tags = input.Tags

	if input.Password != "" {
		pwHash, err := hashPassword(input.Password)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		jar.PasswordHash = &pwHash
	}

	switch {
	case user == nil && input.Expiry.Duration == nil:
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(DurYear), // By default (for anon), we use 1 year expiry
		}
	case input.Expiry.Duration != nil:
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(*input.Expiry.Duration),
		}
	}

	if user != nil {
		userID := user.ID
		jar.UserID = &userID
	}

	err = app.models.ScrollJar.Insert(jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.getJarURI(jar)
	for _, inputScroll := range input.Scrolls {
		scroll := database.Scroll{}
		scroll.JarID = jar.ID
		scroll.Jar = jar
		scroll.Title = inputScroll.Title
		scroll.Format = inputScroll.Format
		scroll.Content = inputScroll.Content
		err = app.models.ScrollJar.InsertScroll(&scroll)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	app.writeJSON(w, http.StatusOK, jar.Jar, nil)
}

func (app *Application) CreateScroll(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	input := spec.ScrollCreationType{}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	jar := database.ScrollJar{}
	jar.ID = id

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

	if !app.verifyJarCreator(id, w, r) {
		app.invalidCredentialsResponse(w, r)
		return
	}

	v := validator.New()
	v.Check(len(input.Content) > 0, "content", "content can't be empty")
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	scroll := database.Scroll{
		Scroll: spec.Scroll{
			Title:   input.Title,
			Content: input.Content,
			Format:  input.Format,
		},
		Jar: &jar,
	}

	err = app.models.ScrollJar.InsertScroll(&scroll)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	app.getScrollURI(&scroll)

	err = app.writeJSON(w, http.StatusOK, scroll.Scroll, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
