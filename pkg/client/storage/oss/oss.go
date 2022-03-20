package oss

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/ciiiii/zookeeper-operator/pkg/client/storage"
	"github.com/ciiiii/zookeeper-operator/pkg/util"
)

var (
	_                      storage.StorageDriver = &StorageDriver{}
	ossTransientErrorCodes                       = []string{"RequestTimeout", "QuotaExceeded.Refresh", "Default", "ServiceUnavailable", "Throttling", "RequestTimeTooSkewed", "SocketException", "SocketTimeout", "ServiceBusy", "DomainNetWorkVisitedException", "ConnectionTimeout", "CachedTimeTooLarge"}
)

type StorageDriver struct {
	*oss.Client
}

func NewClient(endpoint string, accessKey string, secretKey string) (*StorageDriver, error) {
	client, err := oss.New(endpoint, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create new OSS client: %w", err)
	}
	return &StorageDriver{
		Client: client,
	}, err
}

func (s *StorageDriver) Save(bucketName, path, key string) error {
	isDir, err := util.IsDirectory(path)
	if err != nil {
		return err
	}
	bucket, err := s.Bucket(bucketName)
	if err != nil {
		return err
	}
	if isDir {
		if err := putDirectory(bucket, key, path); err != nil {
			return err
		}
	} else {
		if err := putFile(bucket, key, path); err != nil {
			return err
		}
	}
	return nil
}

func (s *StorageDriver) Load(bucketName, key, path string) error {
	bucket, err := s.Bucket(bucketName)
	if err != nil {
		return err
	}
	getErr := bucket.GetObjectToFile(key, path)
	if !isOssErrCode(getErr, "NoSuchKey") {
		return fmt.Errorf("failed to get file: %w", getErr)
	}
	isDir, err := isOssDirectory(bucket, key)
	if err != nil {
		return fmt.Errorf("failed to test if %s/%s is a directory: %w", bucketName, path, err)
	}
	if !isDir {
		return getErr
	}
	if err := getOssDirectory(bucket, key, path); err != nil {
		return fmt.Errorf("failed to get directory: %w", err)
	}
	return nil
}

func getOssDirectory(bucket *oss.Bucket, objectName, path string) error {
	files, err := ListOssDirectory(bucket, objectName)
	if err != nil {
		return err
	}
	for _, f := range files {
		innerName, err := filepath.Rel(objectName, f)
		if err != nil {
			return fmt.Errorf("get Rel path from %s to %s error: %w", f, objectName, err)
		}
		fpath := filepath.Join(path, innerName)
		if strings.HasSuffix(f, "/") {
			err = os.MkdirAll(fpath, 0o700)
			if err != nil {
				return fmt.Errorf("mkdir %s error: %w", fpath, err)
			}
			continue
		}
		dirPath := filepath.Dir(fpath)
		err = os.MkdirAll(dirPath, 0o700)
		if err != nil {
			return fmt.Errorf("mkdir %s error: %w", dirPath, err)
		}

		err = bucket.GetObjectToFile(f, fpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func ListOssDirectory(bucket *oss.Bucket, objectKey string) (files []string, err error) {
	if objectKey != "" {
		if !strings.HasSuffix(objectKey, "/") {
			objectKey += "/"
		}
	}

	pre := oss.Prefix(objectKey)
	marker := oss.Marker("")
	for {
		lor, err := bucket.ListObjects(marker, pre)
		if err != nil {
			return files, err
		}
		for _, obj := range lor.Objects {
			files = append(files, obj.Key)
		}

		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}
	return files, nil
}

func putFile(bucket *oss.Bucket, objectName, path string) error {
	return bucket.PutObjectFromFile(objectName, path)
}

func putDirectory(bucket *oss.Bucket, objectName, dir string) error {
	return filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		nameInDir, err := filepath.Rel(dir, fpath)
		if err != nil {
			return err
		}
		fObjectName := filepath.Join(objectName, nameInDir)
		// create an OSS dir explicitly for every local dir, , including empty dirs.
		if info.Mode().IsDir() {
			// create OSS dir
			if !strings.HasSuffix(fObjectName, "/") {
				fObjectName += "/"
			}
			err = bucket.PutObject(fObjectName, nil)
			if err != nil {
				return err
			}
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		err = putFile(bucket, fObjectName, fpath)
		if err != nil {
			return err
		}
		return nil
	})
}

func isTransientOSSErr(err error) bool {
	if err == nil {
		return false
	}
	if ossErr, ok := err.(oss.ServiceError); ok {
		for _, transientErrCode := range ossTransientErrorCodes {
			if ossErr.Code == transientErrCode {
				return true
			}
		}
	}
	return false
}
