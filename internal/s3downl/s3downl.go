package s3downl

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
)

func GetS3DownloadFunc() func(s3Uri string, path string) error {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	s3Client := s3.NewFromConfig(cfg)

	return func(s3Uri string, path string) error {
		u, err := url.Parse(s3Uri)
		if err != nil {
			return fmt.Errorf("failed to parse s3 uri %s: %w", s3Uri, err)
		}

		if u.Scheme != "s3" {
			return fmt.Errorf("invalid s3 uri scheme: %s", u.Scheme)
		}

		out, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer out.Close()

		log.Printf("Downloading file %s from s3", s3Uri)
		obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(u.Host),
			Key:    aws.String(u.Path[1:]),
		})
		if err != nil {
			return fmt.Errorf("failed to download file %s from s3: %w (host: %s, path: %s)", s3Uri, err, u.Host, u.Path)
		}

		if (obj.ContentType != nil && *obj.ContentType == "application/zstd") ||
			filepath.Ext(u.Path) == ".zst" {

			d, err := zstd.NewReader(obj.Body)
			if err != nil {
				return fmt.Errorf("failed to create zstd reader: %w", err)
			}
			defer d.Close()
			_, err = io.Copy(out, d)
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}

			return nil
		}

		_, err = io.Copy(out, obj.Body)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}

		return nil
	}

}