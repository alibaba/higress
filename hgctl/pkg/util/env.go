package util

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func GetPythonVersion() (string, error) {
	re := regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

	for _, cmd := range []string{"python3", "python"} {
		out, err := exec.Command(cmd, "--version").CombinedOutput()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(out))
		match := re.FindString(version)
		if match != "" {
			return match, nil
		}
	}

	return "", fmt.Errorf("python not found")
}

// compareVersions compares two version strings like "3.11.2" and "3.10".
// Returns:
//
//	 1  if v1 > v2
//	 0  if v1 == v2
//	-1  if v1 < v2
func CompareVersions(v1, v2 string) int {
	// Extract numeric parts only (e.g. "3.12.0b1" → "3.12.0")
	re := regexp.MustCompile(`\d+`)
	nums1 := re.FindAllString(v1, -1)
	nums2 := re.FindAllString(v2, -1)

	maxLen := len(nums1)
	if len(nums2) > maxLen {
		maxLen = len(nums2)
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(nums1) {
			n1, _ = strconv.Atoi(nums1[i])
		}
		if i < len(nums2) {
			n2, _ = strconv.Atoi(nums2[i])
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}
