package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) postUserRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	v.Check(
		len(input.Username) > 0 && len(input.Username) <= 512,
		"username",
		"username must be withing 1-512 charcters",
	)
	v.Check(
		validator.Matches(input.Email, validator.EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)

	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	pwHash, err := hashPassword(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user := &database.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: pwHash,
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrDuplicateUser):
			v.AddError(validator.FieldError{Field: []string{"email"}, Msg: "Duplicate email"})
			app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
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
			if err == nil {
				app.logError(r, err)
			}
		}
	}, "Registration Mail")
	app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
}
