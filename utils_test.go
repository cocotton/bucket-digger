package main

import "testing"

func TestConvertSize(t *testing.T) {
	var tests = []struct {
		sizeByte int64
		sizeUnit string
		expected float64
		err      bool
	}{
		{
			sizeByte: 1024,
			sizeUnit: "kb",
			expected: 1.024,
			err:      false,
		},
	}

	for _, test := range tests {
		result := convertSize(test.sizeByte, test.sizeUnit)
		if result != test.expected {
			t.Errorf("convertSize(): FAILED, Expected '%v' - Received '%v'", test.expected, result)
		}
	}
}

func TestValidateSizeUnitFlag(t *testing.T) {
	var tests = []struct {
		sizeUnit string
		err      bool
	}{
		{
			sizeUnit: "kb",
			err:      false,
		},
		{
			sizeUnit: "xx",
			err:      true,
		},
	}

	for _, test := range tests {
		err := validateSizeUnitFlag(test.sizeUnit)
		if err != nil && test.err == false {
			t.Errorf("validateSizeUnitFlag(): FAILED, Expected no error - Received: %v", err)
		} else if err == nil && test.err {
			t.Errorf("validateSizeUnitFlag(): FAILED, Expected an error - Received: %v", err)
		}
	}
}

func TestValidateFilterFlag(t *testing.T) {
	var tests = []struct {
		filter string
		err    bool
	}{
		{
			filter: "name",
			err:    false,
		},
		{
			filter: "STORAGECLASSES",
			err:    false,
		},
		{
			filter: "badfilter",
			err:    true,
		},
	}

	for _, test := range tests {
		err := validateFilterFlag(test.filter)
		if err != nil && test.err == false {
			t.Errorf("validateFilterFlag(): FAILED, Expected no error - Received: %v", err)
		} else if err == nil && test.err {
			t.Errorf("validateFilterFlag(): FAILED, Expected an error - Received: %v", err)
		}
	}
}

func TestValidateSortFlag(t *testing.T) {
	var tests = []struct {
		sortFlag string
		err      bool
	}{
		{
			sortFlag: "region",
			err:      false,
		},
		{
			sortFlag: "SIZE",
			err:      false,
		},
		{
			sortFlag: "badSortFlag",
			err:      true,
		},
	}

	for _, test := range tests {
		err := validateSortFlag(test.sortFlag)
		if err != nil && test.err == false {
			t.Errorf("validateSortFlag(): FAILED, Expected no error - Received: %v", err)
		} else if err == nil && test.err {
			t.Errorf("validateSortFlag(): FAILED, Expected an error - Received: %v", err)
		}
	}
}

func TestValidateCostPeriodFlag(t *testing.T) {
	var tests = []struct {
		costPeriod int
		err        bool
	}{
		{
			costPeriod: 30,
			err:        false,
		},
		{
			costPeriod: 400,
			err:        true,
		},
		{
			costPeriod: -1,
			err:        true,
		},
	}

	for _, test := range tests {
		err := validateCostPeriodFlag(test.costPeriod)
		if err != nil && test.err == false {
			t.Errorf("validateCostPeriodFlag(): FAILED, Expected no error - Received: %v", err)
		} else if err == nil && test.err {
			t.Errorf("validateCostPeriodFlag(): FAILED, Expected an error - Received: %v", err)
		}
	}
}

func TestValidateWorkersFlag(t *testing.T) {
	var tests = []struct {
		workers int
		err     bool
	}{
		{
			workers: 10,
			err:     false,
		},
		{
			workers: 0,
			err:     true,
		},
		{
			workers: -10,
			err:     true,
		},
	}

	for _, test := range tests {
		err := validateWorkersFlag(test.workers)
		if err != nil && test.err == false {
			t.Errorf("validateWorkersFlag(): FAILED, Expected no error - Received: %v", err)
		} else if err == nil && test.err {
			t.Errorf("validateWorkersFlag(): FAILED, Expected an error - Received: %v", err)
		}
	}
}

func TestFormatStorageClasses(t *testing.T) {
	var emptyFloat float64
	var tests = []struct {
		storageClasses map[string]float64
		expected       string
	}{
		{
			storageClasses: map[string]float64{"STANDARD": float64(10)},
			expected:       "STANDARD(10.0%) ",
		},
		{
			storageClasses: map[string]float64{"STANDARD": float64(10), "ONEZONE_IA": float64(66.666)},
			expected:       "STANDARD(10.0%) ONEZONE_IA(66.7%) ",
		},
		{
			storageClasses: map[string]float64{"STANDARD": emptyFloat},
			expected:       "STANDARD(0.0%) ",
		},
		{
			storageClasses: map[string]float64{},
			expected:       "",
		},
	}

	for _, test := range tests {
		result := formatStorageClasses(test.storageClasses)
		if result != test.expected {
			t.Errorf("formatStorageClasses(): FAILED, Expected: '%v' - Received: '%v'", test.expected, result)
		}
	}
}
