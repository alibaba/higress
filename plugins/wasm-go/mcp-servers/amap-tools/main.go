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

package main

import (
	"amap-tools/tools"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/server"
)

func main() {}

func init() {
	amapServer := &server.MCPServer{}
	server.Load(server.AddMCPServer(
		"amap-tools",
		amapServer.AddMCPTool("maps_geo", &tools.GeoRequest{}).
			AddMCPTool("maps_bicycling", &tools.BicyclingRequest{}).
			AddMCPTool("maps_direction_transit_integrated", &tools.TransitIntegratedRequest{}).
			AddMCPTool("maps_ip_location", &tools.IPLocationRequest{}).
			AddMCPTool("maps_weather", &tools.WeatherRequest{}).
			AddMCPTool("maps_direction_driving", &tools.DrivingRequest{}).
			AddMCPTool("maps_around_search", &tools.AroundSearchRequest{}).
			AddMCPTool("maps_search_detail", &tools.SearchDetailRequest{}).
			AddMCPTool("maps_regeocode", &tools.ReGeocodeRequest{}).
			AddMCPTool("maps_text_search", &tools.TextSearchRequest{}).
			AddMCPTool("maps_distance", &tools.DistanceRequest{}).
			AddMCPTool("maps_direction_walking", &tools.WalkingRequest{}),
	))
}
