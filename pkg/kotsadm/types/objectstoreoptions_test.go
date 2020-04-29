package types

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestObjectStoreValidateAndHydrate(t *testing.T) {
	tests := []struct {
		name         string
		inputOptions *StorageOptions
		wantErr      string
		wantOut      *StorageOptions
	}{
		{
			name: "unsupported object store",
			inputOptions: &StorageOptions{
				ObjectStoreType: "aws",
			},
			wantErr: "unsupported object store type: aws",
		},
		{
			name: "minio and defaults from flags gets hydrated with internal defaults",
			inputOptions: &StorageOptions{
				ObjectStoreType: "internal",
				BucketInPath:    true,
			},
			wantOut: &StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "",
				SecretAccessKey: "",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
		},
		{
			name: "external object store valid",
			inputOptions: &StorageOptions{
				ObjectStoreType: "external",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				BucketName:      "some-bucket",
				Endpoint:        "s3.amazonaws.com",
				BucketInPath:    true,
			},
		},
		{
			name: "external without key",
			inputOptions: &StorageOptions{
				ObjectStoreType: "external",
				AccessKeyID:     "",
				SecretAccessKey: "secret",
				BucketName:      "some-bucket",
				Endpoint:        "s3.amazonaws.com",
				BucketInPath:    true,
			},
			wantErr: `when object store is "external", each of object-store-access-key-id, object-store-secret-access-key, object-store-bucket-name must be set`,
		},
		{
			name: "external without secret",
			inputOptions: &StorageOptions{
				ObjectStoreType: "external",
				AccessKeyID:     "key",
				SecretAccessKey: "",
				BucketName:      "some-bucket",
				Endpoint:        "s3.amazonaws.com",
				BucketInPath:    true,
			},
			wantErr: `when object store is "external", each of object-store-access-key-id, object-store-secret-access-key, object-store-bucket-name must be set`,
		},
		{
			name: "external without endpoint is valid",
			inputOptions: &StorageOptions{
				ObjectStoreType: "external",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				BucketName:      "some-bucket",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			config, err := NewObjectStoreConfig(*test.inputOptions)
			if test.wantErr != "" {
				req.EqualError(err, test.wantErr)
			} else {
				req.Nil(err)
			}

			if test.wantOut != nil {
				req.Equal(test.wantOut.ObjectStoreType, config.options.ObjectStoreType)
				req.Equal(test.wantOut.BucketName, config.options.BucketName)
				req.Equal(test.wantOut.Endpoint, config.options.Endpoint)
				req.Equal(test.wantOut.BucketInPath, config.options.BucketInPath)

				// don't check equality because we're using UUID.New() to generate in some cases,
				// and I don't feel like mocking it, but these should *always* be set
				req.NotEmpty(config.options.AccessKeyID)
				req.NotEmpty(config.options.SecretAccessKey)
			}
		})
	}
}

func TestObjectStoreToSecretData(t *testing.T) {
	tests := []struct {
		name         string
		inputOptions StorageOptions
		wantOut      map[string][]byte
	}{
		{
			name: "convert to secret data",
			inputOptions: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "abcd",
				SecretAccessKey: "efgh",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
			wantOut: map[string][]byte{
				"accesskey":      []byte("abcd"),
				"secretkey":      []byte("efgh"),
				"endpoint":       []byte("http://kotsadm-minio:9000"),
				"bucketname":     []byte("kotsadm"),
				"bucket-in-path": []byte("true"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			data := MustGetObjectStoreConfig(test.inputOptions).ToSecretData()
			req.Equal(test.wantOut, data)
		})
	}
}

func TestObjectStoreLoadSecretData(t *testing.T) {
	tests := []struct {
		name         string
		inputOptions StorageOptions
		inputSecret  map[string][]byte
		wantOut      StorageOptions
		wantErr      string
	}{
		{
			name: "empty Secret",
			inputOptions: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "abcd",
				SecretAccessKey: "efgh",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
			inputSecret: map[string][]byte{},
			wantOut: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "abcd",
				SecretAccessKey: "efgh",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
		},
		{
			name:         "error if can't parse bool",
			inputOptions: StorageOptions{},
			inputSecret: map[string][]byte{
				"bucket-in-path": []byte("no thank you"),
			},
			wantErr: "parse bucket-in-path key of secretData: strconv.ParseBool: parsing \"no thank you\": invalid syntax",
			wantOut: StorageOptions{},
		},
		{
			name: "load data",
			inputOptions: StorageOptions{
			},
			inputSecret: map[string][]byte{
				"type": []byte("internal"),
				"accesskey": []byte("123"),
				"secretkey": []byte("456"),
				"bucketname": []byte("kotsadm"),
				"endpoint": []byte("http://kotsadm-minio:9000"),
				"bucket-in-path": []byte("false"),
			},
			wantOut: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "123",
				SecretAccessKey: "456",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    false,
			},
		},
		{
			name: "overwrite defaults from older version of secret",
			inputOptions: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "some-uuid",
				SecretAccessKey: "some-other-uuid",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
			inputSecret: map[string][]byte{
				"accesskey": []byte("123"),
				"secretkey": []byte("456"),
			},
			wantOut: StorageOptions{
				ObjectStoreType: "internal",
				AccessKeyID:     "123",
				SecretAccessKey: "456",
				BucketName:      "kotsadm",
				Endpoint:        "http://kotsadm-minio:9000",
				BucketInPath:    true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			err := MustGetObjectStoreConfig(test.inputOptions).LoadSecretData(test.inputSecret)
			if test.wantErr != "" {
				req.EqualError(err, test.wantErr)
			}
			req.Equal(test.wantOut, test.inputOptions)
		})
	}
}
