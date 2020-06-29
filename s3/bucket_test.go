package s3

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type mockS3Client struct {
	s3iface.S3API
}

func (m *mockS3Client) ListBuckets(input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	bucket1Name := "bucket1"
	bucket1CreationDate := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	bucket1 := s3.Bucket{
		CreationDate: &bucket1CreationDate,
		Name:         &bucket1Name,
	}

	bucket2Name := "bucket2"
	bucket2CreationDate := time.Date(2020, time.February, 10, 12, 0, 0, 0, time.UTC)

	bucket2 := s3.Bucket{
		CreationDate: &bucket2CreationDate,
		Name:         &bucket2Name,
	}

	buckets := []*s3.Bucket{&bucket1, &bucket2}

	return &s3.ListBucketsOutput{Buckets: buckets}, nil
}

func TestListBuckets(t *testing.T) {
	var expectedBuckets = []*Bucket{
		{time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC), "bucket1"},
		{time.Date(2020, time.February, 10, 12, 0, 0, 0, time.UTC), "bucket2"},
	}

	mockClient := &mockS3Client{}

	buckets, err := ListBuckets(mockClient)

	if err != nil {
		t.Errorf("ListBuckets(): FAILED, expected no errors but received '%v'", err)
	} else {
		if len(buckets) != 2 {
			t.Errorf("ListBuckets(): FAILED, expected 2 buckets but received '%v'", len(buckets))
		} else {
			if *expectedBuckets[0] != *buckets[0] {
				t.Errorf("ListBuckets(): FAILED, expected '%v' but received '%v'", *expectedBuckets[0], *buckets[0])
			} else if *expectedBuckets[1] != *buckets[1] {
				t.Errorf("ListBuckets(): FAILED, expected '%v' but received '%v'", *expectedBuckets[1], *buckets[1])
			} else {
				t.Logf("ListBuckets(): PASSED")
			}
		}
	}

}
