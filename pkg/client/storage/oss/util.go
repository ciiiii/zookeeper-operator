package oss

import (
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func isOssErrCode(err error, code string) bool {
	if serr, ok := err.(oss.ServiceError); ok {
		if serr.Code == code {
			return true
		}
	}
	return false
}

func isOssDirectory(bucket *oss.Bucket, objectName string) (bool, error) {
	if objectName == "" {
		return true, nil
	}
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	rst, err := bucket.ListObjects(oss.Prefix(objectName), oss.MaxKeys(1))
	if err != nil {
		return false, err
	}
	if len(rst.CommonPrefixes)+len(rst.Objects) > 0 {
		return true, nil
	}
	return false, nil
}
