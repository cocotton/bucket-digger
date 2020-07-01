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

// validFilterFlags is a slice containing the valid filters that can be passed as cli arguments with '-filter'
var validFilterFlags = []string{"name", "storageclasses"}

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

// validateGroupByFlag validates the provided group
func validateGroupByFlag(group string) error {
	validGroups := []string{"region"}
	for _, validGroup := range validGroups {
		if strings.ToLower(group) == validGroup {
			return nil
		}
	}
	return fmt.Errorf("Error - '%v' is not a valid '-group' value", group)
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

// formatStorageClasses takes all the storage classes as well as their usage statistics and build a string containing this information
func formatStorageClasses(storageClasses map[string]float64) string {
	b := new(bytes.Buffer)
	for class, value := range storageClasses {
		fmt.Fprintf(b, "%s(%.1f%%) ", class, value)
	}
	return b.String()
}
