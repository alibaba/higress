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

package config

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeGraphQLConfig_SetGql(t *testing.T) {
	tests := []struct {
		name          string
		gql           string
		wantVariables []Variable
		wantErr       error
	}{
		{
			name:    "empty gql",
			gql:     "",
			wantErr: errors.New("gql can't be empty"),
		},
		{
			name:          "no params",
			gql:           "query",
			wantVariables: []Variable{},
			wantErr:       nil,
		},
		{
			name:    "four params",
			gql:     "query ($owner:String $num:Float! $int : Int! $boolean : Boolean  )",
			wantErr: nil,
			wantVariables: []Variable{
				{
					name:  "owner",
					typ:   StringType,
					blank: true,
				},
				{
					name:  "num",
					typ:   FloatType,
					blank: false,
				},
				{
					name:  "int",
					typ:   IntType,
					blank: false,
				},
				{
					name:  "boolean",
					typ:   BooleanType,
					blank: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DeGraphQLConfig{}
			err := d.SetGql(tt.gql)
			assert.Equal(t, tt.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantVariables, d.variables)
		})
	}
}

func TestDeGraphQLConfig_ParseGqlFromUrl(t *testing.T) {

	tests := []struct {
		name    string
		gql     string
		url     string
		want    string
		wantErr error
	}{
		{
			name:    "empty url",
			gql:     "query ($owner:String! $name:String!)",
			url:     "",
			want:    "",
			wantErr: errors.New("request url can't be empty"),
		},

		{
			name:    "no params",
			gql:     "query HeroNameQuery {\n  hero {\n    name\n  }\n}",
			url:     "/api?owner=a",
			want:    "{\"query\":\"query HeroNameQuery {\\n  hero {\\n    name\\n  }\\n}\"}",
			wantErr: nil,
		},

		{
			name:    "one string variable",
			gql:     "query FetchSomeIDQuery($someId: String!) {\n  human(id: $someId) {\n    name\n  }\n}",
			url:     "/api?someId=a",
			want:    "{\"query\":\"query FetchSomeIDQuery($someId: String!) {\\n  human(id: $someId) {\\n    name\\n  }\\n}\",\"variables\":{\"someId\":\"a\"}}",
			wantErr: nil,
		},

		{
			name:    "multi variables",
			gql:     "query FetchSomeIDQuery($someId: String! $num: Int $price: Float! $need:Boolean!) {\n  human(id: $someId) {\n    name\n  }\n}",
			url:     "/api?someId=a&num=10&price=12.0&need=false&hee=1",
			want:    "{\"query\":\"query FetchSomeIDQuery($someId: String! $num: Int $price: Float! $need:Boolean!) {\\n  human(id: $someId) {\\n    name\\n  }\\n}\",\"variables\":{\"someId\":\"a\",\"num\":10,\"price\":12.0,\"need\":false}}",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DeGraphQLConfig{}
			d.SetGql(tt.gql)
			body, err := d.ParseGqlFromUrl(tt.url)
			assert.Equal(t, tt.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, body)
		})
	}
}

func TestDeGraphQLConfig_SetEndpoint(t *testing.T) {

	tests := []struct {
		name     string
		endPoint string
		wantErr  error
		want     string
	}{
		{
			name:     "empty endpoint",
			endPoint: "",
			wantErr:  nil,
			want:     "/graphql",
		},
		{
			name:     "empty endpoint with blank",
			endPoint: "   ",
			wantErr:  nil,
			want:     "/graphql",
		},

		{
			name:     "with value",
			endPoint: " /graphql2 ",
			wantErr:  nil,
			want:     "/graphql2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DeGraphQLConfig{}
			err := d.SetEndpoint(tt.endPoint)
			assert.Equal(t, tt.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, d.endpoint)
		})
	}
}
