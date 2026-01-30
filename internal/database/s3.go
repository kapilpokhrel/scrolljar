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
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func (bucket *S3Bucket) DeleteBatch(toDelete []types.ObjectIdentifier) ([]string, error) {
	errKeys := make([]string, 0)
	output, err := bucket.Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket.cfg.BucketName),
		Delete: &types.Delete{
			Objects: toDelete,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		return nil, nil
	}
	for _, error := range output.Errors {
		errKeys = append(errKeys, *error.Key)
	}
	return errKeys, nil
}

type AvilKeyIterator struct {
	p    *s3.ListObjectsV2Paginator
	page []types.Object
	i    int
}

func (bucket *S3Bucket) NewAvilKeyIterator(ctx context.Context) *AvilKeyIterator {
	return &AvilKeyIterator{
		p: s3.NewListObjectsV2Paginator(bucket.Client, &s3.ListObjectsV2Input{
			Bucket: &bucket.cfg.BucketName,
		}),
	}
}

func (it *AvilKeyIterator) Next(ctx context.Context) (string, bool, error) {
	if it.i < len(it.page) {
		key := *it.page[it.i].Key
		it.i++
		return key, true, nil
	}

	if !it.p.HasMorePages() {
		return "", false, nil
	}

	page, err := it.p.NextPage(ctx)
	if err != nil {
		return "", false, err
	}

	it.page = page.Contents
	it.i = 0

	return it.Next(ctx)
}
