package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type envelope map[string]any

const (
	BaseURI string = "https://scrolljar.com"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func verifyHashPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (app *Application) getJarURI(jar *database.ScrollJar) {
	jar.URI = fmt.Sprintf("%s/jar/%s", BaseURI, jar.ID)
}

func (app *Application) getScrollURI(scroll *database.Scroll) {
	scroll.URI = fmt.Sprintf("%s/scroll/%s", BaseURI, scroll.ID)
}

func (app *Application) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	maps.Copy(w.Header(), headers)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func (app *Application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	if err == nil {
		err = decoder.Decode(&struct{}{})
		if !errors.Is(err, io.EOF) {
			return errors.New("body contains more than one json payload")
		}
		return nil
	}

	// handle error
	var (
		syntaxError           *json.SyntaxError
		unmarshalTypeError    *json.UnmarshalTypeError
		invalidUnmarshalError *json.InvalidUnmarshalError
		maxBytesError         *http.MaxBytesError
	)

	switch {
	case errors.Is(err, io.EOF):
		return errors.New("body is empty")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("body contains badly formed JSON")
	case errors.As(err, &maxBytesError):
		return fmt.Errorf("body exceeds limit of %d bytes", maxBytesError.Limit)
	case errors.As(err, &syntaxError):
		return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
	case errors.As(err, &unmarshalTypeError):
		return fmt.Errorf("body contains incorrect json type for field %s, (at character %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
	case errors.As(err, &invalidUnmarshalError):
		panic(err)
	default:
		return err
	}
}

func (app *Application) backgroundTask(fn func(), taskname string) {
	app.wg.Go(func() {
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%s, %v", taskname, err))
			}
		}()
		fn()
	})
}

func generateToken(uid int64, scope string, expiryDuration time.Duration) (string, *database.Token) {
	tokenText := rand.Text()
	tokenHash := sha256.Sum256([]byte(tokenText))
	token := database.Token{
		TokenHash: tokenHash[:],
		UserID:    uid,
		Scope:     scope,
		ExpiresAt: pgtype.Timestamptz{
			Time: time.Now().Add(expiryDuration),
		},
	}
	return tokenText, &token
}

func (app *Application) verifyJarCreator(jarID string, w http.ResponseWriter, r *http.Request) bool {
	user := app.contextGetUser(r)
	jar := database.ScrollJar{}
	jar.ID = jarID

	err := app.models.ScrollJar.Get(&jar)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
		default:
			app.serverErrorResponse(w, r, err)
		}
		return false
	}

	if *jar.UserID != user.ID {
		return false
	}
	return true
}
