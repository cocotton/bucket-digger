package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cocotton/bucket-digger/s3"
)

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main() {
	// Initialize AWS session in the provided region
	session, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		exitErrorf("Unable to initialize the AWS session, %v", err)
	}

	// Create S3 service client
	client := awss3.New(session)

	// List the buckets in the S3 service client's region
	buckets, err := s3.ListBuckets(client)
	if err != nil {
		exitErrorf("Unable to list the buckets, %v", err)
	}

	for _, bucket := range buckets {
		fmt.Printf("Bucket: %v, Created: %v\n", bucket.Name, bucket.CreationDate)
	}
}
