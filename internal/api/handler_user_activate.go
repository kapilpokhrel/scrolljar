package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) putUserActivationHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	v.Check(
		len(input.Token) > 0,
		"token",
		"token must not be empty",
	)
	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	tokenHash := sha256.Sum256([]byte(input.Token))

	token := &database.Token{
		TokenHash: tokenHash[:],
	}

	err = app.models.Token.GetTokenByHash(token)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			v.AddError(validator.FieldError{Field: []string{"token"}, Msg: "token doesn't exist"})
			app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user := &database.User{
		ID: token.UserID,
	}
	err = app.models.Users.GetByID(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user.Activated = true
	err = app.models.Users.Update(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Token.DeleteByHash(token)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"message": fmt.Sprintf("%s (%s) activated", user.Username, user.Email)}, nil)
}
