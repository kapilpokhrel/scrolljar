package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"

	"github.com/julienschmidt/httprouter"
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

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (app *Application) readIDParam(r *http.Request) string {
	params := httprouter.ParamsFromContext(r.Context())
	return params.ByName("id")
}

func (app *Application) getJarURI(jar *database.ScrollJar) {
	jar.URI = fmt.Sprintf("%s/jar/%s", BaseURI, jar.ID)
}

func (app *Application) getScrollURI(scroll *database.Scroll) {
	scroll.URI = fmt.Sprintf("%s/scroll/%s", BaseURI, scroll.ID)
}

func (app *Application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	jsonString, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	maps.Copy(w.Header(), headers)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonString)
	w.Write([]byte("\n"))
	return nil
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
