package s3downl

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
)

func GetS3DownloadFunc() func(s3Url string, path string) error {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	s3Client := s3.NewFromConfig(cfg)

	return func(s3Url string, path string) error {
		u, err := url.Parse(s3Url)
		if err != nil {
			return fmt.Errorf("failed to parse s3 url %s: %w", s3Url, err)
		}

		if u.Scheme != "https" {
			return fmt.Errorf("invalid s3 url scheme: %s", u.Scheme)
		}

		// Extract bucket from host, assuming format bucket.s3.region.amazonaws.com
		hostParts := strings.Split(u.Host, ".")
		if len(hostParts) < 3 || hostParts[1] != "s3" {
			return fmt.Errorf("invalid s3 url host format: %s", u.Host)
		}
		bucket := hostParts[0]
		key := strings.TrimPrefix(u.Path, "/")

		out, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer out.Close()

		log.Printf("Downloading file %s from s3", s3Url)
		obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("failed to download file %s from s3: %w (bucket: %s, key: %s)", s3Url, err, bucket, key)
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
