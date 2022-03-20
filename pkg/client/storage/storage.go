package storage

type StorageDriver interface {
	Save(bucket, path, key string) error
	Load(bucket, key, path string) error
}
