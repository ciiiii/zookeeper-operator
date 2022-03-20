package helper

import (
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"

	"github.com/ciiiii/zookeeper-operator/pkg/client/storage/oss"
)

func restoreAction(host string) {
	for _, mountPath := range strings.Split(restoreDir, ",") {
		targetPath := filepath.Join(mountPath, dataDir)
		ossClient, err := oss.NewClient(endpoint, accessKey, secretKey)
		if err != nil {
			klog.Fatal(err)
		}
		klog.Infof("start restoring %s from %s:%s", targetPath, bucket, objectKey)
		if err := ossClient.Load(bucket, objectKey, targetPath); err != nil {
			klog.Fatal(err)
		}
		klog.Infof("restore %s from %s:%s success", targetPath, bucket, objectKey)
	}
}
