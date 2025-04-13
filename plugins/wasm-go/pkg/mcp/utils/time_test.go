package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeZoneConversion(t *testing.T) {
	// Create a fixed test time in UTC
	utcTime := time.Date(2025, 4, 13, 9, 30, 0, 0, time.UTC)
	fmt.Printf("Original UTC time: %v\n", utcTime)

	// Test cases for timezone conversion
	testCases := []struct {
		name             string
		fromTZ           string
		toTZ             string
		useStandardTime  bool
		expectedHourDiff int // Expected hour difference from UTC
	}{
		{"UTC to Shanghai", "UTC", "Asia/Shanghai", true, 8},
		{"Shanghai to New York", "Asia/Shanghai", "America/New_York", true, -13}, // 8 - 5 = -13 hour difference from UTC
		{"London to Tokyo", "Europe/London", "Asia/Tokyo", true, 9},
		{"Sydney to Los Angeles", "Australia/Sydney", "America/Los_Angeles", true, -18}, // 10 - 8 = -18 hour difference from UTC
		{"DST: New York to London", "America/New_York", "Europe/London", false, 5},      // Using DST offsets
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get timezone info for logging
			fromTZInfo, err := GetTimeZoneInfo(tc.fromTZ)
			if err != nil {
				t.Fatalf("Failed to get source timezone info: %v", err)
			}

			toTZInfo, err := GetTimeZoneInfo(tc.toTZ)
			if err != nil {
				t.Fatalf("Failed to get target timezone info: %v", err)
			}

			// Log timezone information
			fmt.Printf("\nTest case: %s\n", tc.name)
			fmt.Printf("From: %s (STD: %v, DST: %v)\n",
				fromTZInfo.Name, fromTZInfo.STDOffset, fromTZInfo.DSTOffset)
			fmt.Printf("To: %s (STD: %v, DST: %v)\n",
				toTZInfo.Name, toTZInfo.STDOffset, toTZInfo.DSTOffset)

			// Convert time
			convertedTime, err := ConvertTime(utcTime, tc.fromTZ, tc.toTZ, tc.useStandardTime)
			if err != nil {
				t.Fatalf("Time conversion failed: %v", err)
			}

			// Calculate expected time
			var fromOffset, toOffset time.Duration
			if tc.useStandardTime {
				fromOffset = fromTZInfo.STDOffset
				toOffset = toTZInfo.STDOffset
			} else {
				fromOffset = fromTZInfo.DSTOffset
				toOffset = toTZInfo.DSTOffset
			}

			expectedTime := utcTime.Add(-fromOffset).Add(toOffset)

			fmt.Printf("Original time in %s: %v\n", tc.fromTZ, utcTime)
			fmt.Printf("Converted time to %s: %v\n", tc.toTZ, convertedTime)
			fmt.Printf("Expected time: %v\n", expectedTime)

			// Check if the conversion is correct
			if !convertedTime.Equal(expectedTime) {
				t.Errorf("Time conversion incorrect. Got: %v, Expected: %v",
					convertedTime, expectedTime)
			}

			// Verify hour difference
			hourDiff := int(convertedTime.Sub(utcTime).Hours())
			if hourDiff != tc.expectedHourDiff {
				t.Errorf("Hour difference incorrect. Got: %d, Expected: %d",
					hourDiff, tc.expectedHourDiff)
			}
		})
	}
}

func TestListTimeZones(t *testing.T) {
	zones := ListTimeZones()
	if len(zones) == 0 {
		t.Error("Expected non-empty list of timezones")
	}

	// Print first 10 timezones for verification
	fmt.Println("Sample of available timezones:")
	count := 0
	for _, zone := range zones {
		if count >= 10 {
			break
		}
		info, _ := GetTimeZoneInfo(zone)
		fmt.Printf("%s (STD: %v, DST: %v)\n", zone, info.STDOffset, info.DSTOffset)
		count++
	}

	fmt.Printf("Total number of timezones: %d\n", len(zones))
}

func TestExampleConvertTime(t *testing.T) {
	// Create a time in UTC
	utcTime := time.Date(2025, 4, 13, 12, 0, 0, 0, time.UTC)
	fmt.Println("Example time conversion:")

	// Convert from UTC to New York time (using standard time)
	nyTime, err := ConvertTime(utcTime, "UTC", "America/New_York", true)
	if err != nil {
		t.Fatalf("Error converting to New York time: %v", err)
	}

	fmt.Printf("UTC time: %v\n", utcTime)
	fmt.Printf("New York time: %v\n", nyTime)

	// Convert from New York to Tokyo (using standard time)
	tokyoTime, err := ConvertTime(nyTime, "America/New_York", "Asia/Tokyo", true)
	if err != nil {
		t.Fatalf("Error converting to Tokyo time: %v", err)
	}

	fmt.Printf("Tokyo time: %v\n", tokyoTime)

	// Verify the conversions
	if nyTime.Hour() != 7 { // 12 - 5 = 7 (UTC - New York STD offset)
		t.Errorf("Expected New York hour to be 7, got %d", nyTime.Hour())
	}

	if tokyoTime.Hour() != 21 { // 7 + 14 = 21 (NY hour + (Tokyo - NY) offset)
		t.Errorf("Expected Tokyo hour to be 21, got %d", tokyoTime.Hour())
	}
}
