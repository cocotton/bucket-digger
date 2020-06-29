package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Bucket represents an S3 bucket with added information compared to the github.com/aws/aws-sdk-go/service/s3.Bucket object
type Bucket struct {
	CreationDate time.Time
	Name         string
	Region       string
}

// ListBuckets lists and returns the buckets in the S3 client's region
//  TODO - Would it be possible to cast the s3.Bucket instead of creating whole new objects?
func ListBuckets(client s3iface.S3API) ([]*Bucket, error) {
	var buckets []*Bucket

	result, err := client.ListBuckets(nil)
	if err != nil {
		return nil, err
	}

	for _, b := range result.Buckets {
		buckets = append(buckets, &Bucket{
			CreationDate: *b.CreationDate,
			Name:         *b.Name,
		})
	}

	return buckets, nil
}

// GetBucketRegion gets the bucket's region
func (b *Bucket) GetBucketRegion(client s3iface.S3API) (string, error) {
	ctx := context.Background()
	region, err := s3manager.GetBucketRegionWithClient(ctx, client, b.Name)
	if err != nil {
		return "", err
	}

	return region, nil
}
