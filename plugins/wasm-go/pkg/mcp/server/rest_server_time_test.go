package server

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	template "github.com/higress-group/gjson_template"
)

func TestDateInZoneTemplateFunc(t *testing.T) {
	// Get the template functions
	funcs := templateFuncs()

	// Get the dateInZone function
	dateInZoneFunc, ok := funcs["dateInZone"]
	if !ok {
		t.Fatal("dateInZone function not found in template functions")
	}

	// Convert the function to the expected type
	dateInZone, ok := dateInZoneFunc.(func(string, interface{}, string, ...bool) string)
	if !ok {
		t.Fatal("dateInZone function has unexpected type")
	}

	t.Run("time.Now() to Shanghai", func(t *testing.T) {
		// Get current time in UTC
		nowUTC := time.Now().UTC()
		// Calculate expected time in Shanghai (UTC+8)
		expectedTime := nowUTC.Add(8 * time.Hour)
		expected := expectedTime.Format("2006-01-02 15:04:05")

		// Test with time.Now()
		result := dateInZone("2006-01-02 15:04:05", nowUTC, "Asia/Shanghai", true)

		if result != expected {
			t.Errorf("Expected date %q, got %q", expected, result)
		}
	})

	t.Run("fmt.Sprint(time.Now()) to Shanghai", func(t *testing.T) {
		// Get current time in UTC
		nowUTC := time.Now().UTC()
		// Calculate expected time in Shanghai (UTC+8)
		expectedTime := nowUTC.Add(8 * time.Hour)
		expected := expectedTime.Format("2006-01-02 15:04:05")

		// Test with fmt.Sprint(time.Now())
		result := dateInZone("2006-01-02 15:04:05", fmt.Sprint(nowUTC), "Asia/Shanghai", true)

		if result != expected {
			t.Errorf("Expected date %q, got %q", expected, result)
		}
	})
}

func TestDateInZoneTemplateUsage(t *testing.T) {
	// Create a template that uses the dateInZone function with pipeline syntax
	tmpl, err := template.New("test").Funcs(templateFuncs()).Parse(`
		Current time in Shanghai: {{ dateInZone "2006-01-02 15:04:05" now "Asia/Shanghai" }}
		Current time in New York: {{ dateInZone "2006-01-02 15:04:05" now "America/New_York" }}
		Current time in Tokyo: {{ dateInZone "Jan 02, 2006 at 15:04:05" now "Asia/Tokyo" }}
	`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, []byte("{}")); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	// Get the result
	result := buf.String()

	// Verify that the current time in various timezones is present
	// We can't check the exact value, but we can check that it contains today's date
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(result, today) {
		t.Errorf("Expected result to contain today's date %q, but it doesn't.\nResult: %s", today, result)
	}
}
