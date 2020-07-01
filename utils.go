package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
)

// sizeMap contains the power of 1000 used to convert a byte value into another format, for example a kilobyte
var sizeMap = map[string]float64{
	"b":  math.Pow(1000, 0),
	"kb": math.Pow(1000, 1),
	"mb": math.Pow(1000, 2),
	"gb": math.Pow(1000, 3),
	"tb": math.Pow(1000, 4),
	"pb": math.Pow(1000, 5),
	"eb": math.Pow(1000, 6),
}

// validFilterFlags is a slice containing the valid filter flags that can be passed as cli arguments with '-filter'
var validFilterFlags = []string{"name", "storageclasses"}

// validSortFlags is a slice containing the valid sorting flags that can be passed as cli auguments with '-sort'
var validSortFlags = []string{"name", "region", "size", "files", "created", "modified", "cost"}

// exitErrorf receives an error string as well as any additional arguments, prints them all to Stderr and exit with code 1
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

// printErrorf receives an error string as well as any additional arguments and prints them all to Stderr
func printErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

// convertSize converts a byte size into another format, for example a kilobyte, returning only the value and not the format code
func convertSize(sizeBytes int64, sizeUnit string) float64 {
	return float64(sizeBytes) / sizeMap[sizeUnit]
}

// validateSizeUnitFlag validates that the provided sizeUnit exists in the sizeMap map and is valid
func validateSizeUnitFlag(sizeUnit string) error {
	if _, exists := sizeMap[sizeUnit]; exists {
		return nil
	}

	return fmt.Errorf("Error - '%v' is not a valid '-unit' value", sizeUnit)
}

// validateFilterFlag validates that the provided filter exists in the validFilterFlags slice
func validateFilterFlag(filter string) error {
	for _, validFilter := range validFilterFlags {
		if strings.ToLower(filter) == validFilter {
			return nil
		}
	}
	return fmt.Errorf("Error - '%v' is not a valid '-filter' value", filter)
}

// validateSortFlag validates that the provided sort flag exists in the validSortFlags slice
func validateSortFlag(sortFlag string) error {
	for _, validSortFlag := range validSortFlags {
		if strings.ToLower(sortFlag) == validSortFlag {
			return nil
		}
	}
	return fmt.Errorf("Error - '%v' is not a valid '-sort' value", sortFlag)
}

// validateCostPeriodFlag validates that the provided costPeriod is between 1 and 365
func validateCostPeriodFlag(costPeriod int) error {
	if costPeriod > 365 || costPeriod < 1 {
		return fmt.Errorf("Error - '%v' is not a valid '-costperiod' value, it must be between 1 and 365", costPeriod)
	}
	return nil
}

// formatStorageClasses takes all the storage classes as well as their usage statistics and build a string containing this information
func formatStorageClasses(storageClasses map[string]float64) string {
	b := new(bytes.Buffer)
	for class, value := range storageClasses {
		fmt.Fprintf(b, "%s(%.1f%%) ", class, value)
	}
	return b.String()
}
