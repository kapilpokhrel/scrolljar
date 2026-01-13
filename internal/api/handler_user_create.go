package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) CreateUser(w http.ResponseWriter, r *http.Request) {
	input := spec.Registration{}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(
		len(input.Username) > 0 && len(input.Username) <= 512,
		"username",
		"username must be withing 1-512 charcters",
	)
	v.Check(
		validator.Matches(string(input.Email), validator.EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)

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

	err = app.models.Users.Insert(user)
	if err != nil {
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

	err = app.models.Token.Insert(token)
	if err != nil {
		// TODO
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
			if err != nil {
				app.logError(r, err)
			}
		}
	}, "Registration Mail")
	app.writeJSON(w, http.StatusOK, user.User, nil)
}
