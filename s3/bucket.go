package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Bucket represents an S3 bucket with added information compared to the github.com/aws/aws-sdk-go/service/s3.Bucket object
type Bucket struct {
	CreationDate time.Time
	ObjectCount  int
	LastModified time.Time
	Name         string
	Region       string
	SizeBytes    int64
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

// GetBucketObjectsMetrics gets the number of objects in the bucket
func (b *Bucket) GetBucketObjectsMetrics(client s3iface.S3API) error {
	params := &s3.ListObjectsInput{
		Bucket:  aws.String(b.Name),
		MaxKeys: aws.Int64(400),
	}
	fmt.Println(params.GoString())

	var objects []*s3.Object
	var sizeBytes int64
	var lastModified time.Time

	err := client.ListObjectsPages(params,
		func(page *s3.ListObjectsOutput, last bool) bool {
			for _, obj := range page.Contents {
				objects = append(objects, obj)
				sizeBytes += aws.Int64Value(obj.Size)
				if obj.LastModified.After(lastModified) {
					lastModified = *obj.LastModified
				}
			}
			return true
		},
	)
	if err != nil {
		return err
	}

	b.ObjectCount = len(objects)
	b.SizeBytes = sizeBytes
	b.LastModified = lastModified

	return nil
}
