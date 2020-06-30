package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cocotton/bucket-digger/s3"
)

const defaultRegion = "us-east-1"

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main() {
	// Initialize AWS session in the provided region
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(defaultRegion)},
	)
	if err != nil {
		exitErrorf("Unable to initialize the AWS session, %v", err)
	}

	// Create S3 service client
	client := awss3.New(sess)

	// List the buckets in the S3 service client's region
	buckets, err := s3.ListBuckets(client)
	if err != nil {
		exitErrorf("Unable to list the buckets, %v", err)
	}

	clientMap := make(map[string]*awss3.S3)
	clientMap[defaultRegion] = client

	// Add the bucket's region to each bucket objects
	for _, bucket := range buckets {
		region, err := bucket.GetBucketRegion(client)
		if err != nil {
			exitErrorf("Unable to fetch the region for bucket %v", bucket.Name)
		}
		bucket.Region = region

		// Create a client for a region not found in clientMap
		if _, ok := clientMap[region]; !ok {

			sess, err = session.NewSession(&aws.Config{
				Region: aws.String(region)},
			)
			if err != nil {
				exitErrorf("Unable to initialize the AWS session, %v", err)
			}

			// Create S3 service client
			client = awss3.New(sess)

			clientMap[region] = client
		}
	}

	// Add the bucket's object count to each buckets
	for _, bucket := range buckets {
		err = bucket.GetBucketObjectsMetrics(clientMap[bucket.Region])
		if err != nil {
			exitErrorf("Unable to get the objects metrics for bucket %v", bucket.Name)
		}
	}

}
