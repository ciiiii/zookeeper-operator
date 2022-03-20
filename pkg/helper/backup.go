package helper

import (
	"fmt"
	"path/filepath"

	"k8s.io/klog"

	"github.com/ciiiii/zookeeper-operator/pkg/client/storage/oss"
)

func backupAction(host string) {
	targetPath := filepath.Join(backupDir, dataDir)
	objectKey := fmt.Sprintf("%s/%s", objectKey, host)
	ossClient, err := oss.NewClient(endpoint, accessKey, secretKey)
	if err != nil {
		klog.Fatal(err)
	}
	klog.Infof("start backuping %s to %s:%s", targetPath, bucket, objectKey)
	if err := ossClient.Save(bucket, targetPath, objectKey); err != nil {
		klog.Fatal(err)
	}
	klog.Infof("backup %s to %s:%s success", targetPath, bucket, objectKey)
}
