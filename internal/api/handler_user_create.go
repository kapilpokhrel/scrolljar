package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
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
	user := &database.User{}
	user.Username = input.Username
	user.Email = string(input.Email)
	user.PasswordHash = pwHash

	tx, err := app.models.ScrollJar.GetTx(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	if err := app.models.Users.InsertTx(r.Context(), tx, user); err != nil {
		switch {
		case errors.Is(err, database.ErrDuplicateUser):
			v.AddError(spec.FieldError{Field: []string{"email"}, Msg: "Duplicate email"})
			app.validationErrorResponse(w, r, spec.ValidationError(*v))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	tokenText, token := generateToken(user.ID, database.ScopeActivation, time.Minute*5)

	if err := app.models.Token.InsertTx(r.Context(), tx, token); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	userData := struct {
		ID    int64
		Token string
		Email string
	}{
		ID:    user.ID,
		Token: tokenText,
		Email: user.Email,
	}
	app.backgroundTask(func() {
		for i := 1; i <= 3; i++ {
			err = app.mailer.Send(user.Email, "user_verify.html", userData)
			if err == nil {
				return
			}
			app.logError(r, err)
		}
	}, "Registration Mail")
	app.writeJSON(w, http.StatusOK, user.User, nil)
}
