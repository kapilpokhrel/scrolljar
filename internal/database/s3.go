package database

import (
	"context"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3CFG struct {
	BucketName string
}

type S3Bucket struct {
	cfg    S3CFG
	Client *s3.Client
}

func NewS3Bucket(cfg S3CFG) (*S3Bucket, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsCfg)
	return &S3Bucket{cfg: cfg, Client: s3Client}, nil
}

func (bucket *S3Bucket) GetScrollFetchURL(scroll *Scroll) (string, error) {
	jarID := scroll.JarID
	scrollID := scroll.ID

	key := filepath.Join(jarID, scrollID)
	presignClient := s3.NewPresignClient(bucket.Client)

	fetchURL, err := presignClient.PresignGetObject(
		context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket.cfg.BucketName),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(time.Minute*3),
	)
	return fetchURL.URL, err
}

func (bucket *S3Bucket) StreamingUpload(reader io.Reader, key string) (*manager.UploadOutput, error) {
	uploader := manager.NewUploader(bucket.Client)

	output, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket.cfg.BucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		return nil, err
	}
	return output, err
}
