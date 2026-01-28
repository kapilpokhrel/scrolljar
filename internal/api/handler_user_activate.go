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

	token := &database.Token{
		TokenHash: tokenHash[:],
	}

	if err := app.models.Token.GetTokenByHash(r.Context(), token); err != nil {
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
	if err := app.models.Users.GetByID(r.Context(), user); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user.Activated = true

	tx, err := app.models.ScrollJar.GetTx(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	if err := app.models.Users.UpdateTx(r.Context(), tx, user); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Token.DeleteByHashTx(r.Context(), tx, token); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payload := spec.Message{
		Message: fmt.Sprintf("%s (%s) activated", user.Username, user.Email),
	}

	if err := app.writeJSON(w, http.StatusOK, payload, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
