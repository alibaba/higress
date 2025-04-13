package utils

import (
	"bufio"
	"embed"
	"fmt"
	"strings"
	"time"
)

// Embed the timezone data file
//
//go:embed tz.data
var tzFS embed.FS

// TimeZoneInfo stores offset information for a timezone
type TimeZoneInfo struct {
	Name      string        // Timezone name
	STDOffset time.Duration // Standard Time offset from UTC
	DSTOffset time.Duration // Daylight Saving Time offset from UTC
}

// Global map to store timezone information
var timeZoneMap map[string]TimeZoneInfo

// init function to parse timezone data during initialization
func init() {
	timeZoneMap = make(map[string]TimeZoneInfo)

	// Read and parse the embedded tz.data file
	tzData, err := tzFS.ReadFile("tz.data")
	if err != nil {
		panic(fmt.Sprintf("Failed to read timezone data: %v", err))
	}

	scanner := bufio.NewScanner(strings.NewReader(string(tzData)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) >= 3 {
			name := parts[0]
			stdOffset, err := parseOffset(parts[1])
			if err != nil {
				fmt.Printf("Warning: Failed to parse STD offset for %s: %v\n", name, err)
				continue
			}

			dstOffset, err := parseOffset(parts[2])
			if err != nil {
				fmt.Printf("Warning: Failed to parse DST offset for %s: %v\n", name, err)
				continue
			}

			timeZoneMap[name] = TimeZoneInfo{
				Name:      name,
				STDOffset: stdOffset,
				DSTOffset: dstOffset,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("Error scanning timezone data: %v", err))
	}
}

// parseOffset converts a string offset (like "+08:00" or "-05:30") to a time.Duration
func parseOffset(offsetStr string) (time.Duration, error) {
	// Handle special case for UTC
	if offsetStr == "UTC" || offsetStr == "GMT" {
		return 0, nil
	}

	// Ensure the offset string has a sign
	if !strings.HasPrefix(offsetStr, "+") && !strings.HasPrefix(offsetStr, "−") && !strings.HasPrefix(offsetStr, "-") {
		return 0, fmt.Errorf("invalid offset format: %s", offsetStr)
	}

	// Replace Unicode minus sign with ASCII minus
	offsetStr = strings.ReplaceAll(offsetStr, "−", "-")

	// Split the offset into hours and minutes
	parts := strings.Split(offsetStr[1:], ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid offset format: %s", offsetStr)
	}

	// Parse hours and minutes
	var hours, minutes int
	_, err := fmt.Sscanf(parts[0], "%d", &hours)
	if err != nil {
		return 0, fmt.Errorf("invalid hours in offset: %s", offsetStr)
	}

	_, err = fmt.Sscanf(parts[1], "%d", &minutes)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in offset: %s", offsetStr)
	}

	// Calculate the total offset in seconds
	totalSeconds := hours*3600 + minutes*60

	// Apply the sign
	if strings.HasPrefix(offsetStr, "-") {
		totalSeconds = -totalSeconds
	}

	return time.Duration(totalSeconds) * time.Second, nil
}

// ConvertTime converts a time.Time to the specified timezone
// If useStandardTime is true, it uses the standard time offset, otherwise it uses DST offset
func ConvertTime(t time.Time, fromTimezone, toTimezone string, useStandardTime bool) (time.Time, error) {
	// Get the source timezone info
	fromTZ, ok := timeZoneMap[fromTimezone]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown source timezone: %s", fromTimezone)
	}

	// Get the target timezone info
	toTZ, ok := timeZoneMap[toTimezone]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown target timezone: %s", toTimezone)
	}

	// Determine which offsets to use based on the useStandardTime flag
	fromOffset := fromTZ.STDOffset
	toOffset := toTZ.STDOffset
	if !useStandardTime {
		fromOffset = fromTZ.DSTOffset
		toOffset = toTZ.DSTOffset
	}

	// First convert to UTC by removing the source timezone offset
	utcTime := t.Add(-fromOffset)

	// Then convert to the target timezone by adding its offset
	targetTime := utcTime.Add(toOffset)

	return targetTime, nil
}

// GetTimeZoneInfo returns the timezone information for a given timezone name
func GetTimeZoneInfo(timezone string) (TimeZoneInfo, error) {
	info, ok := timeZoneMap[timezone]
	if !ok {
		return TimeZoneInfo{}, fmt.Errorf("unknown timezone: %s", timezone)
	}
	return info, nil
}

// ListTimeZones returns a list of all available timezone names
func ListTimeZones() []string {
	zones := make([]string, 0, len(timeZoneMap))
	for zone := range timeZoneMap {
		zones = append(zones, zone)
	}
	return zones
}
