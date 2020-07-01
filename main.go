package main

import (
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cheynewallace/tabby"
	"github.com/cocotton/bucket-digger/s3"
)

const defaultRegion = "us-east-1"

func main() {
	// Initialize the cli flags
	var filter, regex, sortasc, sortdes, sizeUnit string
	var costPeriod, workers int

	flag.IntVar(&costPeriod, "costperiod", 30, "The period (in days) over which to calculate the cost of the bucket (e.g. from 30 days ago up to today). Max value: 365")
	flag.StringVar(&filter, "filter", "", "The field to filter on. Possible values: "+strings.Join(validFilterFlags, ", "))
	flag.StringVar(&regex, "regex", "", "The regex to be applied on the filter")
	flag.StringVar(&sortasc, "sortasc", "", "The field to sort (ascending) the output by. Possible values: "+strings.Join(validSortFlags, ", "))
	flag.StringVar(&sortdes, "sortdes", "", "The field to sort (descending) the output by. Possible values: "+strings.Join(validSortFlags, ", "))
	flag.StringVar(&sizeUnit, "unit", "mb", "Unit used to display a bucket's size. Possible values: b, kb, mb, gb, tb, pb, eb")
	flag.IntVar(&workers, "workers", 10, "The number of workers used to fetch the data from AWS")
	flag.Parse()

	// Validate the '-unit' flag
	err := validateSizeUnitFlag(sizeUnit)
	if err != nil {
		exitErrorf(err.Error())
	}

	// Validate the '-costperiod' flag
	err = validateCostPeriodFlag(costPeriod)
	if err != nil {
		exitErrorf(err.Error())
	}

	// Validate the '-workers' flag
	err = validateWorkersFlag(workers)
	if err != nil {
		exitErrorf(err.Error())
	}

	// Make sure the '-filter' and '-regex' flags are provided together or not at all
	// If both flags are provided, validate the '-filter' one and make sure the '-regex' one compiles
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

	// Make sure only one of the '-sortasc' and '-sortdes' flags are provided or none of them
	// If one of the flags is provided, validare it
	if len(sortasc) > 0 || len(sortdes) > 0 {
		if len(sortasc) > 0 && len(sortdes) > 0 {
			exitErrorf("Error - cannot pass both -sortasc and -sortdes flags at the same time")
		} else if len(sortasc) > 0 {
			err = validateSortFlag(sortasc)
		} else if len(sortdes) > 0 {
			err = validateSortFlag(sortdes)
		}
		if err != nil {
			exitErrorf(err.Error())
		}
	}

	// Initialize an AWS session in the defaultRegion
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(defaultRegion)},
	)
	if err != nil {
		exitErrorf("Unable to initialize the AWS session, %v", err)
	}

	// Initialize the S3 client in the defaultRegion
	s3Client := awss3.New(sess)
	// Initialize the cost explorer client in the defaultRegion
	cExplorerClient := costexplorer.New(sess)

	// List all the S3 buckets
	buckets, err := s3.ListBuckets(s3Client)
	if err != nil {
		exitErrorf("Unable to list the buckets, %v", err)
	}

	// Make a map containing an S3 client for every bucket's region
	// We need to use a client with the same region as the bucket's region to be able to fetch the bucket's objects
	clientMap := make(map[string]*awss3.S3)
	clientMap[defaultRegion] = s3Client

	// Make the channel from which the workers will fetch the butckets they need to process
	bucketChan := make(chan *s3.Bucket, len(buckets))

	// Make the slice thay will be filled with the filtered buckets
	filteredBuckets := make([]*s3.Bucket, 0)

	// Initialize a waitgroup that will wait for all the workers to be done working
	var wg sync.WaitGroup

	// Start all the workers
	for i := 1; i <= workers; i++ {
		// Increment the waitgroup and launch a worker into its own goroutine
		wg.Add(1)
		go func() {
			// Decrement the waitgroup when the worker is done working
			defer wg.Done()

			// Loop over the bucket (job) channel to get the buckets to process
			for bucket := range bucketChan {
				// Check if the name filter's regex matches the current bucket's name
				// Skip the bucket if it does not
				if strings.ToLower(filter) == "name" {
					if !compiledRegex.Match([]byte(bucket.Name)) {
						continue
					}
				}

				// Set the bucket's region attribute
				// Skip the bucket if an error is returned. This is because without its region, we might not be able to fetch its objects and will end up with bad informations
				err := bucket.SetBucketRegion(s3Client)
				if err != nil {
					printErrorf("Unable to get the region for bucket %v, skipping it", bucket.Name)
					continue
				} else {
					// Create a client for a region not found in clientMap and add it in clientMap
					if _, ok := clientMap[bucket.Region]; !ok {
						regionSess, err := session.NewSession(&aws.Config{
							Region: aws.String(bucket.Region)},
						)
						if err != nil {
							printErrorf("Unable to initialize the AWS session, %v", err)
						}
						regionClient := awss3.New(regionSess)
						clientMap[bucket.Region] = regionClient
					}

				}

				// Set the bucket objects metrics (e.g. objects count, total size)
				err = bucket.SetBucketObjectsMetrics(clientMap[bucket.Region])
				if err != nil {
					printErrorf("Unable to get the objects metrics for bucket %v, skipping it", bucket.Name)
				}

				// Check if the current bucket has a storage class matching the regex used to filter the buckets
				// Skip the bucket if it does not
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

				// Set the bucket's cost over the provided period (e.g. 30 days)
				err = bucket.SetBucketCostOverPeriod(cExplorerClient, costPeriod)
				if err != nil {
					printErrorf("Error - Unable to get cost for bucket: %v, error: %v", bucket.Name, err)
				}

				// Add the bucket to the list that will be used to output the results in the console
				filteredBuckets = append(filteredBuckets, bucket)
			}
		}()
	}

	// Add the buckets to the bucket (job) channel
	for _, bucket := range buckets {
		bucketChan <- bucket
	}
	// Close the bucket (job) channel to let the workers know no more job will be added to it
	close(bucketChan)
	// Wait for the workers to be done working
	wg.Wait()

	// Sort the bucket list, either ascending or descending, according to the cli flag
	if len(sortasc) > 0 {
		switch sortasc {
		case "name":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Name < filteredBuckets[j].Name })
		case "region":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Region < filteredBuckets[j].Region })
		case "size":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].SizeBytes < filteredBuckets[j].SizeBytes })
		case "files":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].ObjectCount < filteredBuckets[j].ObjectCount })
		case "created":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].CreationDate.Before(filteredBuckets[j].CreationDate) })
		case "modified":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].LastModified.Before(filteredBuckets[j].LastModified) })
		case "cost":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Cost < filteredBuckets[j].Cost })
		}
	} else if len(sortdes) > 0 {
		switch sortdes {
		case "name":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Name > filteredBuckets[j].Name })
		case "region":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Region > filteredBuckets[j].Region })
		case "size":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].SizeBytes > filteredBuckets[j].SizeBytes })
		case "files":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].ObjectCount > filteredBuckets[j].ObjectCount })
		case "created":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].CreationDate.After(filteredBuckets[j].CreationDate) })
		case "modified":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].LastModified.After(filteredBuckets[j].LastModified) })
		case "cost":
			sort.SliceStable(filteredBuckets, func(i, j int) bool { return filteredBuckets[i].Cost > filteredBuckets[j].Cost })
		}
	}

	// Output the buckets to the terminal
	t := tabby.New()
	t.AddHeader("NAME", "REGION", "COST $USD("+strconv.Itoa(costPeriod)+"days)", "TOTAL SIZE ("+strings.ToUpper(sizeUnit)+")", "NUMBER OF FILES", "STORAGE CLASSES", "CREATED ON", "LAST MODIFIED")
	for _, bucket := range filteredBuckets {
		t.AddLine(
			bucket.Name,
			bucket.Region,
			fmt.Sprintf("%f", bucket.Cost),
			fmt.Sprintf("%.2f", convertSize(bucket.SizeBytes, sizeUnit)),
			bucket.ObjectCount,
			formatStorageClasses(bucket.StorageClassesStats),
			bucket.CreationDate.Format("02-01-2006"),
			bucket.LastModified.Format("02-01-2006"),
		)
	}
	t.Print()
}
