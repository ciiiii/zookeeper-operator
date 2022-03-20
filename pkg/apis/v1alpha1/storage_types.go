package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type StroageLocation struct {
	S3  *S3Location  `json:"s3,omitempty"`
	OSS *OSSLocation `json:"oss,omitempty"`
}

type GenericStroageLocation struct {
	Endpoint  string
	Bucket    string
	Key       string
	SecretKey *corev1.SecretKeySelector
	AccessKey *corev1.SecretKeySelector
}

func (location *StroageLocation) GetStroageLocation() (*GenericStroageLocation, error) {
	switch {
	case location.S3 != nil:
		return location.S3.getStroageLocation()
	case location.OSS != nil:
		return location.OSS.getStroageLocation()
	default:
		return nil, fmt.Errorf("no stroage location found")
	}
}

type S3Location struct {
	S3Bucket `json:",inline"`
	// +optional
	Key string `json:"key,omitempty"`
}

func (location *S3Location) getStroageLocation() (*GenericStroageLocation, error) {
	return nil, fmt.Errorf("s3 stroageLocation not implemented")
}

type S3Bucket struct {
	Endpoint        string                    `json:"endpoint,omitempty"`
	Bucket          string                    `json:"bucket,omitempty"`
	Region          string                    `json:"region,omitempty"`
	AccessKeySecret *corev1.SecretKeySelector `json:"accessKeySecret,omitempty"`
	SecretKeySecret *corev1.SecretKeySelector `json:"secretKeySecret,omitempty"`
}

type OSSLocation struct {
	OSSBucket `json:",inline"`
	// +optional
	Key string `json:"key"`
}

func (location *OSSLocation) getStroageLocation() (*GenericStroageLocation, error) {
	if location.OSSBucket.Endpoint == "" {
		return nil, fmt.Errorf("oss stroageLocation endpoint is required")
	}
	if location.OSSBucket.Bucket == "" {
		return nil, fmt.Errorf("oss stroageLocation bucket is required")
	}
	if location.OSSBucket.AccessKeySecret == nil {
		return nil, fmt.Errorf("oss stroageLocation accessKeySecret is required")
	}
	if location.OSSBucket.SecretKeySecret == nil {
		return nil, fmt.Errorf("oss stroageLocation secretKeySecret is required")
	}
	return &GenericStroageLocation{
		Endpoint:  location.Endpoint,
		Bucket:    location.Bucket,
		Key:       location.Key,
		AccessKey: location.AccessKeySecret,
		SecretKey: location.SecretKeySecret,
	}, nil
}

type OSSBucket struct {
	// +kubebuilder:required
	Endpoint string `json:"endpoint,omitempty"`
	// +kubebuilder:required
	Bucket string `json:"bucket,omitempty"`
	// +kubebuilder:required
	AccessKeySecret *corev1.SecretKeySelector `json:"accessKeySecret,omitempty"`
	// +kubebuilder:required
	SecretKeySecret          *corev1.SecretKeySelector `json:"secretKeySecret,omitempty"`
	CreateBucketIfNotPresent bool                      `json:"createBucketIfNotPresent,omitempty"`
}
