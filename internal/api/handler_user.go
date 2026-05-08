package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

func (app *Application) CreateUser(w http.ResponseWriter, r *http.Request) {
	if err := app.createUser(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) createUser(w http.ResponseWriter, r *http.Request) error {
	input := spec.RegistrationInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	v := input.Validate()
	if !v.Valid() {
		return errValidation(spec.ValidationError(*v))
	}

	pwHash, err := hashPassword(input.Password)
	if err != nil {
		return err
	}

	user, tokenText, err := app.store.CreateUserWithActivationToken(r.Context(), database.InsertUserParams{
		Username:     input.Username,
		Email:        string(input.Email),
		PasswordHash: pwHash,
	})
	if err != nil {
		if errors.Is(err, database.ErrDuplicateUser) {
			v.AddError(spec.FieldError{Field: []string{"email"}, Msg: "Duplicate email"})
			return errValidation(spec.ValidationError(*v))
		}
		return err
	}

	userData := struct {
		ID    int64
		Token string
		Email string
	}{ID: user.ID, Token: tokenText, Email: user.Email}

	app.backgroundTask(func() {
		for i := 1; i <= 3; i++ {
			if err := app.mailer.Send(user.Email, "user_verify.html", userData); err == nil {
				return
			} else {
				app.logger.Error(err.Error())
			}
		}
	}, "Registration Mail")

	return app.writeJSON(w, http.StatusOK, dbUserToSpec(user), nil)
}

func (app *Application) AuthUser(w http.ResponseWriter, r *http.Request) {
	if err := app.authUser(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) authUser(w http.ResponseWriter, r *http.Request) error {
	input := spec.LoginInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	v := input.Validate()
	if !v.Valid() {
		return errValidation(spec.ValidationError(*v))
	}

	user, err := app.store.GetUserByEmail(r.Context(), string(input.Email))
	if err != nil {
		return dbErr(err)
	}
	if !verifyHashPassword(input.Password, user.PasswordHash) {
		return errInvalidCreds
	}
	if !user.Activated {
		return errInactiveAccount
	}

	tokens, err := app.store.UpsertAuthTokens(r.Context(), user.ID)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.AuthTokens{
		Authorization: spec.Token{Token: tokens.AuthText, Expiry: tokens.AuthExpiry},
		Refresh:       spec.Token{Token: tokens.RefreshText, Expiry: tokens.RefreshExpiry},
	}, nil)
}

func (app *Application) GetUser(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	if err := app.writeJSON(w, http.StatusOK, dbUserToSpec(*user), nil); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) GetUserJars(w http.ResponseWriter, r *http.Request) {
	if err := app.getUserJars(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) getUserJars(w http.ResponseWriter, r *http.Request) error {
	user := app.contextGetUser(r)
	jars, err := app.store.GetJarsByUser(r.Context(), user.ID)
	if err != nil {
		return err
	}
	out := make([]spec.Jar, len(jars))
	for i, j := range jars {
		out[i] = dbJarToSpec(j)
	}
	return app.writeJSON(w, http.StatusOK, out, nil)
}

func (app *Application) ActivateUser(w http.ResponseWriter, r *http.Request) {
	if err := app.activateUser(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) activateUser(w http.ResponseWriter, r *http.Request) error {
	input := spec.ActivationInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	v := input.Validate()
	if !v.Valid() {
		return errValidation(spec.ValidationError(*v))
	}

	tokenHash := sha256.Sum256([]byte(input.Token))
	user, err := app.store.ActivateUser(r.Context(), tokenHash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			v.AddError(spec.FieldError{Field: []string{"token"}, Msg: "token doesn't exist"})
			return errValidation(spec.ValidationError(*v))
		}
		return err
	}

	return app.writeJSON(w, http.StatusOK, spec.Message{
		Message: fmt.Sprintf("%s (%s) activated", user.Username, user.Email),
	}, nil)
}
