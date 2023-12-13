// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package main_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	a := now.Add(time.Duration(7200) * time.Second)
	assert.Equal(t, a.Sub(now).Seconds(), float64(7200))
}
