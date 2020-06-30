package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cheynewallace/tabby"
	"github.com/cocotton/bucket-digger/s3"
)

const defaultRegion = "us-east-1"

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func printErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func validateFlags(sizeUnit string) error {
	validUnits := []string{"b", "kb", "mb", "gb", "tb", "pb", "eb"}
	for _, validUnit := range validUnits {
		if validUnit == strings.ToLower(sizeUnit) {
			return nil
		}
	}

	return errors.New("Wrong size unit provided")
}

func convertSize(sizeBytes int64, sizeUnit string) float64 {
	sizeMap := map[string]float64{
		"b":  1,
		"kb": 1000,
		"mb": math.Pow(1000, 2),
		"gb": math.Pow(1000, 3),
		"tb": math.Pow(1000, 4),
		"pb": math.Pow(1000, 5),
		"eb": math.Pow(1000, 6),
	}
	return float64(sizeBytes) / sizeMap[sizeUnit]
}

func formatStorageClasses(storageClasses map[string]float64) string {
	b := new(bytes.Buffer)
	for class, value := range storageClasses {
		fmt.Fprintf(b, "%s(%.1f%%) ", class, value)
	}
	return b.String()
}

func main() {
	// Initialize the cli flags
	var sizeUnit string
	var workers int
	flag.StringVar(&sizeUnit, "unit", "mb", "The unit used to display a bucket's size - b, kb, mb, gb, tb, pb, eb")
	flag.IntVar(&workers, "workers", 10, "The number of workers digging through S3")
	flag.Parse()

	// Validate the flags
	err := validateFlags(sizeUnit)
	if err != nil {
		exitErrorf(err.Error())
	}

	// Initialize the AWS session in the defaultRegion
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(defaultRegion)},
	)
	if err != nil {
		exitErrorf("Unable to initialize the AWS session, %v", err)
	}

	// Initialize the S3 service client in the defaultRegion
	client := awss3.New(sess)

	// List the buckets in the S3 service client's region
	buckets, err := s3.ListBuckets(client)
	if err != nil {
		exitErrorf("Unable to list the buckets, %v", err)
	}

	// Make a map containing an S3 service client for every buckets regions
	clientMap := make(map[string]*awss3.S3)
	clientMap[defaultRegion] = client

	// Make the channels that will be used by the workers to find out the buckets regions
	b := make(chan *s3.Bucket, len(buckets))
	//bRegion := make(chan *s3.Bucket, len(buckets))
	var wg sync.WaitGroup

	// Warm up the workers
	for i := 1; i <= workers; i++ {
		wg.Add(1)

		go func(c *awss3.S3, id int) {
			defer wg.Done()

			for bucket := range b {
				// Get the bucket's region and add it to its attributes
				err := bucket.GetBucketRegion(client)
				if err != nil {
					printErrorf("Unable to fetch the region for bucket %v, skipping it", bucket.Name)
					continue
				} else {
					// Create a client for a region not found in clientMap
					if _, ok := clientMap[bucket.Region]; !ok {
						sess, err = session.NewSession(&aws.Config{
							Region: aws.String(bucket.Region)},
						)
						if err != nil {
							printErrorf("Unable to initialize the AWS session, %v", err)
						}
						// Create S3 service client
						client = awss3.New(sess)
						clientMap[bucket.Region] = client
					}

				}

				// Get the bucket objects' metrics
				err = bucket.GetBucketObjectsMetrics(clientMap[bucket.Region])
				if err != nil {
					printErrorf("Unable to get the objects metrics for bucket %v, skipping it", bucket.Name)
				}
			}
		}(client, i)
	}

	// Add the buckets to the job channel
	for _, bucket := range buckets {
		b <- bucket
	}
	close(b)
	wg.Wait()

	// Output the buckets to the terminal
	t := tabby.New()
	t.AddHeader("NAME", "REGION", "TOTAL SIZE ("+strings.ToUpper(sizeUnit)+")", "NUMBER OF FILES", "STORAGE CLASSES", "CREATED ON", "LAST MODIFIED")
	for _, bucket := range buckets {

		t.AddLine(bucket.Name,
			bucket.Region,
			fmt.Sprintf("%.2f", convertSize(bucket.SizeBytes, sizeUnit)),
			bucket.ObjectCount,
			formatStorageClasses(bucket.StorageClassesStats),
			bucket.CreationDate,
			bucket.LastModified)
	}
	t.Print()
}
