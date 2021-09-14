// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/integration/s3_storage_client_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.
// +build requires_docker

package integration

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"

	"github.com/grafana/mimir/integration/e2e"
	e2edb "github.com/grafana/mimir/integration/e2e/db"
	s3 "github.com/grafana/mimir/pkg/chunk/aws"
	mimir_s3 "github.com/grafana/mimir/pkg/storage/bucket/s3"
)

func TestS3Client(t *testing.T) {
	s, err := e2e.NewScenario(networkName)
	require.NoError(t, err)
	defer s.Close()

	// Start dependencies.
	minio := e2edb.NewMinio(9000, bucketName)
	require.NoError(t, s.StartAndWaitReady(minio))

	tests := []struct {
		name string
		cfg  s3.S3Config
	}{
		{
			name: "expanded-config",
			cfg: s3.S3Config{
				Endpoint:         minio.HTTPEndpoint(),
				BucketNames:      bucketName,
				S3ForcePathStyle: true,
				Insecure:         true,
				AccessKeyID:      e2edb.MinioAccessKey,
				SecretAccessKey:  e2edb.MinioSecretKey,
			},
		},
		{
			name: "url-config",
			cfg: s3.S3Config{
				S3: flagext.URLValue{
					URL: urlMustParse("http://" + e2edb.MinioAccessKey + ":" + e2edb.MinioSecretKey + "@" + minio.HTTPEndpoint()),
				},
				BucketNames:      bucketName,
				S3ForcePathStyle: true,
			},
		},
		{
			name: "mixed-config",
			cfg: s3.S3Config{
				S3: flagext.URLValue{
					URL: urlMustParse("http://" + minio.HTTPEndpoint()),
				},
				BucketNames:      bucketName,
				S3ForcePathStyle: true,
				AccessKeyID:      e2edb.MinioAccessKey,
				SecretAccessKey:  e2edb.MinioSecretKey,
			},
		},
		{
			name: "config-with-deprecated-sse",
			cfg: s3.S3Config{
				Endpoint:         minio.HTTPEndpoint(),
				BucketNames:      bucketName,
				S3ForcePathStyle: true,
				Insecure:         true,
				AccessKeyID:      e2edb.MinioAccessKey,
				SecretAccessKey:  e2edb.MinioSecretKey,
				SSEEncryption:    true,
			},
		},
		{
			name: "config-with-sse-s3",
			cfg: s3.S3Config{
				Endpoint:         minio.HTTPEndpoint(),
				BucketNames:      bucketName,
				S3ForcePathStyle: true,
				Insecure:         true,
				AccessKeyID:      e2edb.MinioAccessKey,
				SecretAccessKey:  e2edb.MinioSecretKey,
				SSEConfig: mimir_s3.SSEConfig{
					Type: "SSE-S3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "config-with-deprecated-sse", "config-with-sse-s3":
				t.Skip("TODO: Issue #231")
			}

			client, err := s3.NewS3ObjectClient(tt.cfg)

			require.NoError(t, err)

			ctx := context.Background()
			objectKey := "key-" + tt.name
			obj := []byte{0x01, 0x02, 0x03, 0x04}

			err = client.PutObject(ctx, objectKey, bytes.NewReader(obj))
			require.NoError(t, err)

			readCloser, err := client.GetObject(ctx, objectKey)
			require.NoError(t, err)

			read := make([]byte, 4)
			_, err = readCloser.Read(read)
			if err != io.EOF {
				require.NoError(t, err)
			}

			require.Equal(t, obj, read)
		})
	}
}

func urlMustParse(parse string) *url.URL {
	u, err := url.Parse(parse)
	if err != nil {
		panic(err)
	}

	return u
}
