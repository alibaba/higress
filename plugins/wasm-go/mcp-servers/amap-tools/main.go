// main.go
package main

import (
	"amap-test/server"
	"amap-test/tools"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
    wrapper.SetCtx(
        "amap-test", // Server name
        wrapper.ParseRawConfig(server.ParseFromConfig),
        // wrapper.AddMCPTool("my_tool", tools.MyTool{}), // Register tools
        // Add more tools as needed
        wrapper.AddMCPTool("maps_bicycling", tools.BicyclingRequest{}),
        wrapper.AddMCPTool("maps_geo", tools.GeoRequest{}),
        wrapper.AddMCPTool("maps_direction_transit_integrated", tools.TransitIntegratedRequest{}),
        wrapper.AddMCPTool("maps_ip_location", tools.IPLocationRequest{}),
        wrapper.AddMCPTool("maps_weather", tools.WeatherRequest{}),
        wrapper.AddMCPTool("maps_direction_driving", tools.DrivingRequest{}),
        wrapper.AddMCPTool("maps_around_search", tools.AroundSearchRequest{}),
        wrapper.AddMCPTool("maps_search_detail", tools.SearchDetailRequest{}),
        wrapper.AddMCPTool("maps_regeocode", tools.ReGeocodeRequest{}),
        wrapper.AddMCPTool("maps_text_search", tools.TextSearchRequest{}),
        wrapper.AddMCPTool("maps_distance", tools.DistanceRequest{}),
        wrapper.AddMCPTool("maps_direction_walking", tools.WalkingRequest{}),
    )
}