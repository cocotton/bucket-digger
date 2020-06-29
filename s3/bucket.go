package s3

import (
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// Bucket represents an S3 bucket with added information compared to the github.com/aws/aws-sdk-go/service/s3.Bucket object
type Bucket struct {
	// Date the bucket was created.
	CreationDate time.Time

	// The name of the bucket.
	Name string
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
