package api

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgtype"
	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

var DurYear time.Duration = time.Hour * 25 * 365

func insertJarFromInputType(app *Application, input spec.JarCreation, user *database.User) (*database.ScrollJar, error) {
	jar := &database.ScrollJar{}
	jar.Name = input.Name
	jar.Access = input.Access
	jar.Tags = input.Tags

	if input.Password != "" {
		pwHash, err := hashPassword(input.Password)
		if err != nil {
			return nil, err
		}
		jar.PasswordHash = &pwHash
	}

	switch {
	case user == nil && input.Expiry.Duration == nil:
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(DurYear), // By default (for anon), we use 1 year expiry
		}
	case input.Expiry.Duration != nil:
		jar.ExpiresAt = pgtype.Timestamptz{
			Time: time.Now().Add(*input.Expiry.Duration),
		}
	}

	if user != nil {
		userID := user.ID
		jar.UserID = &userID
	}

	err := app.models.ScrollJar.Insert(jar)
	if err != nil {
		return nil, err
	}
	app.getJarURI(jar)
	return jar, nil
}

func insertScrollFromInputType(app *Application, input spec.ScrollCreation, jarID string, user *database.User) (*database.Scroll, string, error) {
	scroll := database.Scroll{
		Scroll: spec.Scroll{
			Title:  input.Title,
			Format: input.Format,
			JarID:  jarID,
		},
	}

	err := app.models.ScrollJar.InsertScroll(&scroll)
	if err != nil {
		return nil, "", err
	}
	app.getScrollURI(&scroll)

	uploadToken, err := createScrollUploadToken(&scroll, user)
	if err != nil {
		return nil, "", err
	}
	return &scroll, uploadToken, nil
}

func (app *Application) CreateJar(w http.ResponseWriter, r *http.Request) {
	input := spec.JarCreation{}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)
	v := input.Validate(user != nil)
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	jar, err := insertJarFromInputType(app, input, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	createdScrolls := make([]spec.ScrollCreationResponse, 0, len(input.Scrolls))
	for _, inputScroll := range input.Scrolls {
		scroll, uploadToken, err := insertScrollFromInputType(app, inputScroll, jar.ID, user)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			// TODO: NEED TO DO THE CLEANUP
			return
		}
		createdScrolls = append(
			createdScrolls,
			spec.ScrollCreationResponse{Scroll: scroll.Scroll, UploadToken: uploadToken},
		)
	}
	err = app.writeJSON(w, http.StatusOK, spec.JarCreationResponse{
		Jar:     jar.Jar,
		Scrolls: createdScrolls,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) CreateScroll(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	input := spec.ScrollCreationType{}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if !app.verifyJarCreator(id, w, r) {
		app.invalidCredentialsResponse(w, r)
		return
	}

	user := app.contextGetUser(r)
	scroll, uploadToken, err := insertScrollFromInputType(app, input, id, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.writeJSON(
		w,
		http.StatusOK,
		spec.ScrollCreationResponse{Scroll: scroll.Scroll, UploadToken: uploadToken},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) UploadScroll(w http.ResponseWriter, r *http.Request, params spec.UploadScrollParams) {
	token := params.XUploadToken
	scrollID, jarID, userID, err := verifyScrollUploadToken(token)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	scroll := database.Scroll{}
	scroll.ID = scrollID
	app.models.ScrollJar.GetScroll(&scroll)
	if scroll.Uploaded {
		app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "already uploaded"})
		return

	}

	var maxSize int64 = 1 * 1024 * 1024 // For anon user
	if userID >= 0 {
		maxSize = 10 * 1024 * 1024 // 10 MB
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+1)

	key := filepath.Join(jarID, scrollID)
	reader := utf8ValidationReader{r: r.Body}

	uploader := manager.NewUploader(app.s3Client)

	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(app.config.S3.BucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		if errors.Is(err, utf8Err) {
			app.badRequestResponse(w, r, errors.New("invalid text content"))
			return
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			app.entityTooLarge(w, r)
			http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.ScrollJar.SetScrollUpload(&scroll)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrEditConflict):
			app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "edit confict"})
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	app.getScrollURI(&scroll)
	fetchURL, err := app.getScrollFetchURL(&scroll)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(
		w,
		http.StatusOK,
		spec.ScrollFetch{Scroll: scroll.Scroll, FetchURL: fetchURL},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
