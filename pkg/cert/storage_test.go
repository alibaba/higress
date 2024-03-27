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

package cert

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetConfigmapStoreNameByKey(t *testing.T) {
	// Create a fake client for testing
	fakeClient := fake.NewSimpleClientset()
	// Create a new ConfigmapStorage instance for testing
	namespace := "your-namespace"
	storage := &ConfigmapStorage{
		namespace: namespace,
		client:    fakeClient,
	}
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "certificate crt",
			key:      "/certificates/issuerKey/domain.crt",
			expected: "higress-cert-store-certificates-" + fastHash([]byte("issuerKey"+"domain")),
		},
		{
			name:     "certificate meta",
			key:      "/certificates/issuerKey/domain.json",
			expected: "higress-cert-store-certificates-" + fastHash([]byte("issuerKey"+"domain")),
		},
		{
			name:     "certificate key",
			key:      "/certificates/issuerKey/domain.key",
			expected: "higress-cert-store-certificates-" + fastHash([]byte("issuerKey"+"domain")),
		},
		{
			name:     "user key",
			key:      "/users/hello/2",
			expected: "higress-cert-store-default",
		},
		{
			name:     "Empty Key",
			key:      "",
			expected: "higress-cert-store-default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storageName := storage.getConfigmapStoreNameByKey(test.key)
			assert.Equal(t, test.expected, storageName)
		})
	}
}

func TestExists(t *testing.T) {
	// Create a fake client for testing
	fakeClient := fake.NewSimpleClientset()

	// Create a new ConfigmapStorage instance for testing
	namespace := "your-namespace"
	storage, err := NewConfigmapStorage(namespace, fakeClient)
	assert.NoError(t, err)

	// Store a test key
	testKey := "/certificates/issuer1/domain1.crt"
	err = storage.Store(context.Background(), testKey, []byte("test-data"))
	assert.NoError(t, err)

	// Define test cases
	tests := []struct {
		name        string
		key         string
		shouldExist bool
	}{
		{
			name:        "Existing Key",
			key:         "/certificates/issuer1/domain1.crt",
			shouldExist: true,
		},
		{
			name:        "Non-Existent Key1",
			key:         "/certificates/issuer2/domain2.crt",
			shouldExist: false,
		},
		{
			name:        "Non-Existent Key2",
			key:         "/users/hello/a",
			shouldExist: false,
		},
		// Add more test cases as needed
	}

	// Run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists := storage.Exists(context.Background(), test.key)
			assert.Equal(t, test.shouldExist, exists)
		})
	}
}

func TestLoad(t *testing.T) {
	// Create a fake client for testing
	fakeClient := fake.NewSimpleClientset()

	// Create a new ConfigmapStorage instance for testing
	namespace := "your-namespace"
	storage, err := NewConfigmapStorage(namespace, fakeClient)
	assert.NoError(t, err)

	// Store a test key
	testKey := "/certificates/issuer1/domain1.crt"
	testValue := []byte("test-data")
	err = storage.Store(context.Background(), testKey, testValue)
	assert.NoError(t, err)

	// Define test cases
	tests := []struct {
		name        string
		key         string
		expected    []byte
		shouldError bool
	}{
		{
			name:        "Existing Key",
			key:         "/certificates/issuer1/domain1.crt",
			expected:    testValue,
			shouldError: false,
		},
		{
			name:        "Non-Existent Key",
			key:         "/certificates/issuer2/domain2.crt",
			expected:    nil,
			shouldError: true,
		},
		// Add more test cases as needed
	}

	// Run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := storage.Load(context.Background(), test.key)
			if test.shouldError {
				assert.Error(t, err)
				assert.Nil(t, value)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, value)
			}
		})
	}
}

func TestStore(t *testing.T) {
	// Create a fake client for testing
	fakeClient := fake.NewSimpleClientset()

	// Create a new ConfigmapStorage instance for testing
	namespace := "your-namespace"
	storage := ConfigmapStorage{
		namespace: namespace,
		client:    fakeClient,
	}

	// Define test cases
	tests := []struct {
		name                  string
		key                   string
		value                 []byte
		expected              map[string]string
		expectedConfigmapName string
		shouldError           bool
	}{
		{
			name:                  "Store Key with /certificates prefix",
			key:                   "/certificates/issuer1/domain1.crt",
			value:                 []byte("test-data1"),
			expected:              map[string]string{fastHash([]byte("/certificates/issuer1/domain1.crt")): `{"k":"/certificates/issuer1/domain1.crt","v":"dGVzdC1kYXRhMQ=="}`},
			expectedConfigmapName: "higress-cert-store-certificates-" + fastHash([]byte("issuer1"+"domain1")),
			shouldError:           false,
		},
		{
			name:  "Store Key with /certificates prefix (additional data)",
			key:   "/certificates/issuer2/domain2.crt",
			value: []byte("test-data2"),
			expected: map[string]string{
				fastHash([]byte("/certificates/issuer2/domain2.crt")): `{"k":"/certificates/issuer2/domain2.crt","v":"dGVzdC1kYXRhMg=="}`,
			},
			expectedConfigmapName: "higress-cert-store-certificates-" + fastHash([]byte("issuer2"+"domain2")),
			shouldError:           false,
		},
		{
			name:                  "Store Key without /certificates prefix",
			key:                   "/other/path/data.txt",
			value:                 []byte("test-data3"),
			expected:              map[string]string{fastHash([]byte("/other/path/data.txt")): `{"k":"/other/path/data.txt","v":"dGVzdC1kYXRhMw=="}`},
			expectedConfigmapName: "higress-cert-store-default",
			shouldError:           false,
		},
		// Add more test cases as needed
	}

	// Run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := storage.Store(context.Background(), test.key, test.value)
			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check the contents of the ConfigMap after storing
				configmapName := storage.getConfigmapStoreNameByKey(test.key)
				cm, err := fakeClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
				assert.NoError(t, err)

				// Check if the data is as expected
				assert.Equal(t, test.expected, cm.Data)

				// Check if the configmapName is correct
				assert.Equal(t, test.expectedConfigmapName, configmapName)
			}
		})
	}
}

func TestList(t *testing.T) {
	// Create a fake client for testing
	fakeClient := fake.NewSimpleClientset()

	// Create a new ConfigmapStorage instance for testing
	namespace := "your-namespace"
	storage, err := NewConfigmapStorage(namespace, fakeClient)
	assert.NoError(t, err)

	// Store some test data
	// Store some test data
	testKeys := []string{
		"/certificates/issuer1/domain1.crt",
		"/certificates/issuer1/domain2.crt",
		"/certificates/issuer1/domain3.crt", // Added another domain for issuer1
		"/certificates/issuer2/domain4.crt",
		"/certificates/issuer2/domain5.crt",
		"/certificates/issuer3/subdomain1/domain6.crt",            // Two-level subdirectory under issuer3
		"/certificates/issuer3/subdomain1/subdomain2/domain7.crt", // Two more levels under issuer3
		"/other-prefix/key1/file1",
		"/other-prefix/key1/file2",
		"/other-prefix/key2/file3",
		"/other-prefix/key2/file4",
	}

	for _, key := range testKeys {
		err := storage.Store(context.Background(), key, []byte("test-data"))
		assert.NoError(t, err)
	}

	// Define test cases
	tests := []struct {
		name      string
		prefix    string
		recursive bool
		expected  []string
	}{
		{
			name:      "List Certificates (Non-Recursive)",
			prefix:    "/certificates",
			recursive: false,
			expected:  []string{"/certificates/issuer1", "/certificates/issuer2", "/certificates/issuer3"},
		},
		{
			name:      "List Certificates (Recursive)",
			prefix:    "/certificates",
			recursive: true,
			expected:  []string{"/certificates/issuer1/domain1.crt", "/certificates/issuer1/domain2.crt", "/certificates/issuer1/domain3.crt", "/certificates/issuer2/domain4.crt", "/certificates/issuer2/domain5.crt", "/certificates/issuer3/subdomain1/domain6.crt", "/certificates/issuer3/subdomain1/subdomain2/domain7.crt"},
		},
		{
			name:      "List Other Prefix (Non-Recursive)",
			prefix:    "/other-prefix",
			recursive: false,
			expected:  []string{"/other-prefix/key1", "/other-prefix/key2"},
		},

		{
			name:      "List Other Prefix (Non-Recursive)",
			prefix:    "/other-prefix/key1",
			recursive: false,
			expected:  []string{"/other-prefix/key1/file1", "/other-prefix/key1/file2"},
		},
		{
			name:      "List Other Prefix (Recursive)",
			prefix:    "/other-prefix",
			recursive: true,
			expected:  []string{"/other-prefix/key1/file1", "/other-prefix/key1/file2", "/other-prefix/key2/file3", "/other-prefix/key2/file4"},
		},
	}

	// Run tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys, err := storage.List(context.Background(), test.prefix, test.recursive)
			assert.NoError(t, err)
			assert.ElementsMatch(t, test.expected, keys)
		})
	}
}
