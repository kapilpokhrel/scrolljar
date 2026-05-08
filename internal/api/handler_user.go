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
	input := spec.RegistrationInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := input.Validate()
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	pwHash, err := hashPassword(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	user, tokenText, err := app.store.CreateUserWithActivationToken(r.Context(), database.InsertUserParams{
		Username:     input.Username,
		Email:        string(input.Email),
		PasswordHash: pwHash,
	})
	if err != nil {
		if errors.Is(err, database.ErrDuplicateUser) {
			v.AddError(spec.FieldError{Field: []string{"email"}, Msg: "Duplicate email"})
			app.validationErrorResponse(w, r, spec.ValidationError(*v))
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
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

	app.writeJSON(w, http.StatusOK, dbUserToSpec(user), nil)
}

func (app *Application) AuthUser(w http.ResponseWriter, r *http.Request) {
	input := spec.LoginInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := input.Validate()
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	user, err := app.store.GetUserByEmail(r.Context(), string(input.Email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	if !verifyHashPassword(input.Password, user.PasswordHash) {
		app.invalidCredentialsResponse(w, r)
		return
	}
	if !user.Activated {
		app.inactiveAccountResponse(w, r)
		return
	}

	tokens, err := app.store.UpsertAuthTokens(r.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, spec.AuthTokens{
		Authorization: spec.Token{Token: tokens.AuthText, Expiry: tokens.AuthExpiry},
		Refresh:       spec.Token{Token: tokens.RefreshText, Expiry: tokens.RefreshExpiry},
	}, nil)
}

func (app *Application) GetUser(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	if err := app.writeJSON(w, http.StatusOK, dbUserToSpec(*user), nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetUserJars(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	jars, err := app.store.GetJarsByUser(r.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	out := make([]spec.Jar, len(jars))
	for i, j := range jars {
		out[i] = dbJarToSpec(j)
	}
	if err := app.writeJSON(w, http.StatusOK, out, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) ActivateUser(w http.ResponseWriter, r *http.Request) {
	input := spec.ActivationInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := input.Validate()
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	tokenHash := sha256.Sum256([]byte(input.Token))
	user, err := app.store.ActivateUser(r.Context(), tokenHash[:])
	// we don't use handleDBErr here because we want to map "token doesn't exist" to a validation error instead of 404 or 500.
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			v.AddError(spec.FieldError{Field: []string{"token"}, Msg: "token doesn't exist"})
			app.validationErrorResponse(w, r, spec.ValidationError(*v))
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, spec.Message{
		Message: fmt.Sprintf("%s (%s) activated", user.Username, user.Email),
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
