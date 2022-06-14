package storage

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
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
	SourceFilePath string
	LocalPath      string
}

func (s *StorageService) DownloadSourceObjects(submissionId string) ([]*DownloadedObject, error) {
	var files []*DownloadedObject

	config := env.GetEnv()

	tempDirectory, err := os.MkdirTemp("", fmt.Sprintf("%s-", submissionId))
	if err != nil {
		return nil, fmt.Errorf("Failed to create temp dir: %s", err)
	}

	prefix := fmt.Sprintf("%s/%s/", config.SubmissionFolderPrefix, submissionId)

	for object := range s.s3Service.GetObjects(prefix) {
		localPath, err := s.downloadSourceObject(object.Path, tempDirectory)
		if err != nil {
			return nil, err
		}

		files = append(files, &DownloadedObject{
			SourceFilePath: strings.TrimPrefix(object.Path, prefix),
			LocalPath:      localPath,
		})
	}

	return files, nil
}

func (s *StorageService) downloadSourceObject(objectPath string, directory string) (string, error) {
	object, err := s.s3Service.GetObject(objectPath)
	if err != nil {
		return "", err
	}
	defer object.Close()

	localPath := fmt.Sprintf("%s/%s", directory, objectPath)
	logrus.Debugf("Downloading file '%s' to '%s'", objectPath, localPath)
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		return "", err
	}

	f, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %s", err)
	}

	_, err = io.Copy(f, object)
	if err != nil {
		return "", fmt.Errorf("failed to copy object to '%s': %s", localPath, err)
	}

	return localPath, nil
}

func (s *StorageService) UploadTranspiledObject(submissionId string, filename string, content []byte) error {
	config := env.GetEnv()
	objectPath := fmt.Sprintf("%s-results/%s/%s", config.SubmissionFolderPrefix, submissionId, filename)

	logrus.Debugf("Uploading file '%s' to '%s'", filename, objectPath)

	_, err := s.s3Service.PutObject(objectPath, bytes.NewReader(content), int64(len(content)))
	return err
}
