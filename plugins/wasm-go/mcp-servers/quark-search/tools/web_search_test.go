// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestWebSearchInputSchema tests the InputSchema method of WebSearch
// to verify that the JSON schema configuration is correct.
func TestWebSearchInputSchema(t *testing.T) {
	// Create a WebSearch instance
	webSearch := WebSearch{}

	// Get the input schema
	schema := webSearch.InputSchema()

	// Marshal the schema to JSON for better readability
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema to JSON: %v", err)
	}

	// Print the schema
	fmt.Printf("WebSearch InputSchema:\n%s\n", string(schemaJSON))

	// Basic validation to ensure the schema is not empty
	if len(schema) == 0 {
		t.Error("InputSchema returned an empty schema")
	}
}
