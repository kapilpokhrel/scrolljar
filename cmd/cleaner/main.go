// Package cleaner is for the periodic database cleaner binary
package cleaner

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/logger"
)

type cleanerCfg struct {
	DBURL        string
	S3BucketName string
}

func parseFlags() cleanerCfg {
	var cfg cleanerCfg
	flag.StringVar(&cfg.DBURL, "db_url", os.Getenv("SCROLLJAR_DB_URL"), "PostgreSQL URL")
	flag.StringVar(&cfg.S3BucketName, "s3-bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	flag.Parse()
	return cfg
}

func main() {
	log := logger.SetupLogger("cleaner")
	if err := godotenv.Load(); err != nil {
		log.Error("Error loading .env file")
	}

	cfg := parseFlags()

	dbPool, err := database.SetupDB(database.DBCFG{URL: cfg.DBURL})
	if err != nil {
		log.Error(err.Error())
		return
	}
	store := database.NewStore(dbPool)

	s3Bucket, err := database.NewS3Bucket(database.S3CFG{BucketName: cfg.S3BucketName})
	if err != nil {
		log.Error(err.Error())
		return
	}

	const batchSize = 1000
	var batch []types.ObjectIdentifier
	var scrollIDs []string

	ctx := context.Background()
	it := s3Bucket.NewAvilKeyIterator(ctx)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		existing, err := store.GetExistingScrollIDs(ctx, scrollIDs)
		if err != nil {
			log.Error(err.Error())
			return
		}
		existsMap := make(map[string]bool, len(existing))
		for _, id := range existing {
			existsMap[id] = true
		}
		var toDelete []types.ObjectIdentifier
		for _, obj := range batch {
			scrollID := strings.SplitN(*obj.Key, "/", 2)[1]
			if !existsMap[scrollID] {
				toDelete = append(toDelete, obj)
			}
		}
		s3Bucket.DeleteBatch(toDelete)
		batch, scrollIDs = batch[:0], scrollIDs[:0]
	}

	for {
		key, ok, err := it.Next(ctx)
		if err != nil {
			log.Error(err.Error())
			return
		}
		if !ok {
			break
		}

		parts := strings.SplitN(key, "/", 2)
		if len(parts) < 2 {
			continue
		}
		scrollIDs = append(scrollIDs, parts[1])
		batch = append(batch, types.ObjectIdentifier{Key: &key})

		if len(batch) == batchSize {
			flush()
		}
	}
	flush()
}
