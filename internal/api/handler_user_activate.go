package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) ActivateUser(w http.ResponseWriter, r *http.Request) {
	input := spec.ActivationInput{}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := input.Validate()
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
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
			v.AddError(spec.FieldError{Field: []string{"token"}, Msg: "token doesn't exist"})
			app.validationErrorResponse(w, r, spec.ValidationError(*v))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user := &database.User{}
	user.ID = token.UserID
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

	payload := spec.Message{
		Message: fmt.Sprintf("%s (%s) activated", user.Username, user.Email),
	}

	app.writeJSON(w, http.StatusOK, payload, nil)
}
