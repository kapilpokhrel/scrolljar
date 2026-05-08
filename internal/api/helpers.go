package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/spec"
	"golang.org/x/crypto/bcrypt"
)

type envelope map[string]any

const baseURI string = "https://scrolljar.com"

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func verifyHashPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func jarURI(id string) string {
	return fmt.Sprintf("%s/jar/%s", baseURI, id)
}

func scrollURI(id string) string {
	return fmt.Sprintf("%s/scroll/%s", baseURI, id)
}

func dbJarToSpec(jar database.Scrolljar) spec.Jar {
	return spec.Jar{
		ID:        jar.ID,
		Name:      jar.Name.String,
		Access:    spec.JarAccess(jar.Access),
		Tags:      jar.Tags,
		ExpiresAt: jar.ExpiresAt,
		CreatedAt: jar.CreatedAt,
		URI:       jarURI(jar.ID),
	}
}

func dbScrollToSpec(scroll database.Scroll) spec.Scroll {
	return spec.Scroll{
		ID:        scroll.ID,
		JarID:     scroll.JarID,
		Title:     scroll.Title.String,
		Format:    scroll.Format.String,
		CreatedAt: scroll.CreatedAt,
		URI:       scrollURI(scroll.ID),
	}
}

func dbUserToSpec(user database.UserAccount) spec.User {
	return spec.User{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Time,
	}
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

var secretKey = []byte("<SECRET_KEY>")

func createScrollUploadToken(scrollID, jarID string, user *database.UserAccount) (string, error) {
	var userID int64 = -1
	if user != nil && user.Activated {
		userID = user.ID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"scrollID": scrollID,
		"jarID":    jarID,
		"userID":   userID,
		"exp":      time.Now().Add(time.Minute * 5).Unix(),
	})
	return token.SignedString(secretKey)
}

func verifyScrollUploadToken(tokenString string) (scrollID, jarID string, userID int64, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return secretKey, nil
	})
	if err != nil {
		return "", "", -1, err
	}
	if !token.Valid {
		return "", "", -1, fmt.Errorf("invalid token")
	}
	claims := token.Claims.(jwt.MapClaims)
	return claims["scrollID"].(string), claims["jarID"].(string), int64(claims["userID"].(float64)), nil
}

// checkJarPassword returns an error if the jar is private and the supplied password is wrong.
func checkJarPassword(jar database.Scrolljar, password string) error {
	if jar.Access != int16(spec.AccessPrivate) {
		return nil
	}
	if password == "" || !verifyHashPassword(password, jar.PasswordHash.String) {
		return errors.New("invalid jar password")
	}
	return nil
}

// isJarCreator checks whether the authenticated user owns the given jar.
func (app *Application) isJarCreator(r *http.Request, jarID string) (bool, error) {
	user := app.contextGetUser(r)
	ownerID, err := app.store.GetJarOwnerID(r.Context(), jarID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return ownerID.Valid && ownerID.Int64 == user.ID, nil
}

// requireJarCreator writes the appropriate error and returns false if the caller should stop.
func (app *Application) requireJarCreator(w http.ResponseWriter, r *http.Request, jarID string) bool {
	ok, err := app.isJarCreator(r, jarID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return false
	}
	if !ok {
		app.invalidCredentialsResponse(w, r)
		return false
	}
	return true
}

// handleDBErr handles not-found vs server error. Returns true if the handler should stop.
func (app *Application) handleDBErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		app.notFoundResponse(w, r)
	} else {
		app.serverErrorResponse(w, r, err)
	}
	return true
}

// handleDBErrWithConflict is like handleDBErr but also maps ErrEditConflict to 409.
func (app *Application) handleDBErrWithConflict(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, database.ErrEditConflict):
		app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "edit conflict; please try again"})
	case errors.Is(err, pgx.ErrNoRows):
		app.notFoundResponse(w, r)
	default:
		app.serverErrorResponse(w, r, err)
	}
	return true
}

var utf8Err = errors.New("invalid UTF-8")

type utf8ValidationReader struct {
	r io.Reader
}

func (u utf8ValidationReader) Read(p []byte) (int, error) {
	n, err := u.r.Read(p)
	if n > 0 && !utf8.Valid(p[:n]) {
		return 0, utf8Err
	}
	return n, err
}

// jarExpiryFromInput resolves the expiry for a new jar given the input and whether the user is authenticated.
func jarExpiryFromInput(input spec.CreateJarInput, authenticated bool) pgtype.Timestamptz {
	const durYear = time.Hour * 25 * 365
	switch {
	case !authenticated && input.Expiry.Duration == nil:
		return pgtype.Timestamptz{Time: time.Now().Add(durYear), Valid: true}
	case input.Expiry.Duration != nil:
		return pgtype.Timestamptz{Time: time.Now().Add(*input.Expiry.Duration), Valid: true}
	default:
		return pgtype.Timestamptz{}
	}
}
