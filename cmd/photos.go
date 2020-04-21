package main

import (
	"context"
	"io"
	"mime/multipart"
	"time"

	"cloud.google.com/go/storage"
	"github.com/rishabh-bector/BenevolentBitesBack/auth"
	log "github.com/sirupsen/logrus"
)

// Uploads photo from restaurant to Google Cloud Storage DB
func UploadPhotos(fhs []*multipart.FileHeader) ([]string, error) {
	ctx := context.Background()

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error(err)
		return []string{}, err
	}

	// Sets the name for the new bucket.
	bucketName := "benevolentbites-restaurant-photos"

	var attrs []string
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			log.Error(err)
			return []string{}, err
		}

		// Creates new object
		obj := client.Bucket(bucketName).Object(auth.GenerateUUID())

		wc := obj.NewWriter(ctx)

		// Writes photo to bucket object
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()
		if _, err := io.Copy(wc, f); err != nil {
			log.Error(err)
			return []string{}, err
		}
		if err := wc.Close(); err != nil {
			log.Error(err)
			return []string{}, err
		}

		// Sets object to public reading permissions
		if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			log.Error(err)
			return []string{}, err
		}

		// Gets object attributes (including public URL)
		attr, err := obj.Attrs(ctx)
		if err != nil {
			log.Error(err.Error())
			return []string{}, err
		}
		attrs = append(attrs, attr.MediaLink)
	}

	return attrs, nil
}
