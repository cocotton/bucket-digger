package s3

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/costexplorer/costexploreriface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Bucket represents an S3 bucket with added information compared to the github.com/aws/aws-sdk-go/service/s3.Bucket object
type Bucket struct {
	Cost                float64
	CreationDate        time.Time
	ObjectCount         int
	LastModified        time.Time
	Name                string
	Region              string
	SizeBytes           int64
	StorageClassesStats map[string]float64
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

// SetBucketRegion sets the bucket's region
func (b *Bucket) SetBucketRegion(client s3iface.S3API) error {
	ctx := context.Background()
	region, err := s3manager.GetBucketRegionWithClient(ctx, client, b.Name)
	if err != nil {
		return err
	}

	b.Region = region

	return nil
}

// SetBucketObjectsMetrics sets the metrics related to a bucket's objects
func (b *Bucket) SetBucketObjectsMetrics(client s3iface.S3API) error {
	params := &s3.ListObjectsInput{
		Bucket:  aws.String(b.Name),
		MaxKeys: aws.Int64(400),
	}

	var objects []*s3.Object
	var sizeBytes int64
	var lastModified time.Time
	storageClasses := map[string]float64{}

	err := client.ListObjectsPages(params,
		func(page *s3.ListObjectsOutput, last bool) bool {
			for _, obj := range page.Contents {
				objects = append(objects, obj)
				sizeBytes += aws.Int64Value(obj.Size)
				if obj.LastModified.After(lastModified) {
					lastModified = *obj.LastModified
				}
				storageClasses[aws.StringValue(obj.StorageClass)]++
			}
			return true
		},
	)
	if err != nil {
		return err
	}

	for class, count := range storageClasses {
		storageClasses[class] = count / float64(len(objects)) * 100
	}

	b.ObjectCount = len(objects)
	b.SizeBytes = sizeBytes
	b.LastModified = lastModified
	b.StorageClassesStats = storageClasses

	return nil
}

// SetBucketCostOverPeriod sets the bucket's cost from now up to X days ago
func (b *Bucket) SetBucketCostOverPeriod(client costexploreriface.CostExplorerAPI, period int, tag string) error {
	now := time.Now().AddDate(0, 0, 1)
	then := now.AddDate(0, 0, -period)

	param := &costexplorer.GetCostAndUsageInput{
		Filter: &costexplorer.Expression{
			And: []*costexplorer.Expression{
				{
					Dimensions: &costexplorer.DimensionValues{
						Key:    aws.String("SERVICE"),
						Values: []*string{aws.String("Amazon Simple Storage Service")},
					},
				},
				{
					Tags: &costexplorer.TagValues{
						Key:    aws.String(tag),
						Values: []*string{aws.String(b.Name)},
					},
				},
			},
		},
		Granularity: aws.String("MONTHLY"),
		Metrics:     []*string{aws.String("AmortizedCost")},
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(then.Format("2006-01-02")),
			End:   aws.String(now.Format("2006-01-02")),
		},
	}

	results, err := client.GetCostAndUsage(param)
	if err != nil {
		return err
	}

	var cost float64
	for _, result := range results.ResultsByTime {
		amount, _ := strconv.ParseFloat(aws.StringValue(result.Total["AmortizedCost"].Amount), 64)
		cost += amount
	}

	b.Cost = cost

	return nil
}
