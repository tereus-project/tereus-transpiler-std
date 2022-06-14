package storage

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/tereus-project/tereus-go-std/s3"
	"github.com/tereus-project/tereus-transpiler-std/env"
)

type StorageService struct {
	s3Service *s3.S3Service
}

func NewStorageService() (*StorageService, error) {
	config := env.GetEnv()

	s3Service, err := s3.NewS3Service(config.S3Endpoint, config.S3AccessKey, config.S3SecretKey, config.S3Bucket, config.S3HTTPSEnabled)
	if err != nil {
		return nil, err
	}

	return &StorageService{
		s3Service: s3Service,
	}, nil
}

type DownloadedObject struct {
	ObjectPath string
	LocalPath  string
}

func (s *StorageService) DownloadObjects(submissionId string) ([]*DownloadedObject, error) {
	var files []*DownloadedObject

	for object := range s.s3Service.GetObjects(submissionId) {
		localPath, err := s.downloadObject(submissionId, object.Path)
		if err != nil {
			return nil, err
		}

		files = append(files, &DownloadedObject{
			ObjectPath: object.Path,
			LocalPath:  localPath,
		})
	}

	return files, nil
}

func (s *StorageService) downloadObject(submissionId string, filename string) (string, error) {
	config := env.GetEnv()
	objectPath := fmt.Sprintf("%s/%s/%s", config.SubmissionFolderPrefix, submissionId, filename)

	object, err := s.s3Service.GetObject(objectPath)
	if err != nil {
		return "", err
	}
	defer object.Close()

	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-", submissionId))
	if err != nil {
		return "", fmt.Errorf("Failed to create temp dir: %s", err)
	}

	f, err := os.Create(fmt.Sprintf("%s/%s", dir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to create file: %s", err)
	}

	_, err = io.Copy(f, object)
	if err != nil {
		return "", fmt.Errorf("failed to copy object to '%s': %s", f.Name(), err)
	}

	return f.Name(), nil
}

func (s *StorageService) UploadObject(submissionId string, filename string, content []byte) error {
	config := env.GetEnv()
	objectPath := fmt.Sprintf("%s-results/%s/%s", config.SubmissionFolderPrefix, submissionId, filename)

	_, err := s.s3Service.PutObject(objectPath, bytes.NewReader(content), int64(len(content)))
	return err
}
