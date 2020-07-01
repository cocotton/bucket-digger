package main

import (
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cheynewallace/tabby"
	"github.com/cocotton/bucket-digger/s3"
)

const defaultRegion = "us-east-1"

func main() {
	// Initialize the cli flags
	var filter, groupBy, regex, sizeUnit string
	var workers int

	flag.StringVar(&filter, "filter", "", "The field to filter on. Possible values: name, storageclasses")
	flag.StringVar(&groupBy, "group", "", "Group the buckets by - region")
	flag.StringVar(&regex, "regex", "", "The regex to be applied on the filter")
	flag.StringVar(&sizeUnit, "unit", "mb", "Unit used to display a bucket's size - b, kb, mb, gb, tb, pb, eb")
	flag.IntVar(&workers, "workers", 10, "The number of workers digging through S3")
	flag.Parse()

	// Validate the flags
	err := validateSizeUnitFlag(sizeUnit)
	if err != nil {
		exitErrorf(err.Error())
	}

	if len(groupBy) > 0 {
		err = validateGroupByFlag(groupBy)
		if err != nil {
			exitErrorf(err.Error())
		}
	}

	var compiledRegex *regexp.Regexp
	if filter != "" || regex != "" {
		if filter != "" && regex != "" {
			err = validateFilterFlag(filter)
			if err != nil {
				exitErrorf(err.Error())
			}

			compiledRegex, err = regexp.Compile(regex)
			if err != nil {
				exitErrorf("Unable to compile the provided regex, %v", err)
			}
		} else {
			exitErrorf("The -filter and -regex must be used together")
		}
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

	// Make the channel from where the workers will fetch the butckets they need to process
	bucketChan := make(chan *s3.Bucket, len(buckets))

	// Make the slice thay will be filled with the filtered buckets
	filteredBuckets := make([]*s3.Bucket, 0)

	var wg sync.WaitGroup
	// Warm up the workers
	for i := 1; i <= workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for bucket := range bucketChan {
				// Check if the name filter's regex matches the current bucket's name, skip loop if it does not
				if strings.ToLower(filter) == "name" {
					if !compiledRegex.Match([]byte(bucket.Name)) {
						continue
					}
				}

				// Get the bucket's region and add it to its attributes
				err := bucket.GetBucketRegion(client)
				if err != nil {
					printErrorf("Unable to fetch the region for bucket %v, skipping it", bucket.Name)
					continue
				} else {
					// Create a client for a region not found in clientMap
					if _, ok := clientMap[bucket.Region]; !ok {
						regionSess, err := session.NewSession(&aws.Config{
							Region: aws.String(bucket.Region)},
						)
						if err != nil {
							printErrorf("Unable to initialize the AWS session, %v", err)
						}
						// Create S3 service client
						regionClient := awss3.New(regionSess)
						clientMap[bucket.Region] = regionClient
					}

				}

				// Get the bucket objects' metrics
				err = bucket.GetBucketObjectsMetrics(clientMap[bucket.Region])
				if err != nil {
					printErrorf("Unable to get the objects metrics for bucket %v, skipping it", bucket.Name)
				}

				// Check if the current bucket has a storage class matching the regex used to filter the buckets
				if strings.ToLower(filter) == "storageclasses" {
					hasStorageClass := false
					for class := range bucket.StorageClassesStats {
						if compiledRegex.Match([]byte(class)) {
							hasStorageClass = true
						}
					}
					if !hasStorageClass {
						continue
					}
				}

				filteredBuckets = append(filteredBuckets, bucket)
			}
		}()
	}

	// Add the buckets to the job channel
	for _, bucket := range buckets {
		bucketChan <- bucket
	}
	close(bucketChan)
	wg.Wait()

	// Group the buckets by the provided group flag
	if groupBy == "region" {
		sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Region < filteredBuckets[j].Region })
	}

	// Output the buckets to the terminal
	t := tabby.New()
	t.AddHeader("NAME", "REGION", "TOTAL SIZE ("+strings.ToUpper(sizeUnit)+")", "NUMBER OF FILES", "STORAGE CLASSES", "CREATED ON", "LAST MODIFIED")
	for _, bucket := range filteredBuckets {
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
