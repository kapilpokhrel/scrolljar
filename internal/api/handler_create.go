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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

var DurYear time.Duration = time.Hour * 25 * 365

func insertJarFromInputType(ctx context.Context, tx pgx.Tx, app *Application, input spec.CreateJarInput, user *database.User) (*database.ScrollJar, error) {
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

	if err := app.models.ScrollJar.InsertTx(ctx, tx, jar); err != nil {
		return nil, err
	}
	app.getJarURI(jar)
	return jar, nil
}

func insertScrollFromInputType(ctx context.Context, tx pgx.Tx, app *Application, input spec.CreateScrollInput, jarID string, user *database.User) (*database.Scroll, string, error) {
	scroll := database.Scroll{
		Scroll: spec.Scroll{
			Title:  input.Title,
			Format: input.Format,
			JarID:  jarID,
		},
	}

	if err := app.models.ScrollJar.InsertScrollTx(ctx, tx, &scroll); err != nil {
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
	input := spec.CreateJarInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)
	v := input.Validate(user != nil)
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	tx, err := app.models.ScrollJar.GetTx(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	jar, err := insertJarFromInputType(r.Context(), tx, app, input, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	createdScrolls := make([]spec.CreateScrollOutput, 0, len(input.Scrolls))
	for _, inputScroll := range input.Scrolls {
		scroll, uploadToken, err := insertScrollFromInputType(r.Context(), tx, app, inputScroll, jar.ID, user)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		createdScrolls = append(
			createdScrolls,
			spec.CreateScrollOutput{Scroll: scroll.Scroll, UploadToken: uploadToken},
		)
	}

	if err := tx.Commit(r.Context()); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, spec.CreateJarOutput{
		Jar:     jar.Jar,
		Scrolls: createdScrolls,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) CreateScroll(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	input := spec.CreateScrollInput{}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if !app.verifyJarCreator(id, w, r) {
		app.invalidCredentialsResponse(w, r)
		return
	}

	user := app.contextGetUser(r)

	tx, err := app.models.ScrollJar.GetTx(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	scroll, uploadToken, err := insertScrollFromInputType(r.Context(), tx, app, input, id, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(
		w,
		http.StatusOK,
		spec.CreateScrollOutput{Scroll: scroll.Scroll, UploadToken: uploadToken},
		nil,
	); err != nil {
		app.serverErrorResponse(w, r, err)
		return
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
	app.models.ScrollJar.GetScroll(r.Context(), &scroll)
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

	if err := app.models.ScrollJar.SetScrollUpload(r.Context(), &scroll); err != nil {
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

	if err := app.writeJSON(
		w,
		http.StatusOK,
		spec.ScrollFetch{Scroll: scroll.Scroll, FetchURL: fetchURL},
		nil,
	); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
