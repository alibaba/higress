---
title: IP Geolocation
keywords: [higress,geo ip]
description: IP Geolocation Plugin Configuration Reference
---
## Function Description
The `geo-ip` plugin allows querying geographical location information based on the user's IP address, and then passes this geographical information to subsequent plugins through request attributes and newly added request headers.

## Runtime Properties
Plugin Execution Phase: `Authentication Phase`  
Plugin Execution Priority: `440`  

## Configuration Fields
| Name            | Data Type    | Requirement | Default Value      | Description  |
| --------        | -----------  | ----------- | ------------------ | ------------ |
|  ip_protocol    |  string      |  No         |   ipv4             |  Optional values: 1. ipv4: Only queries geographical location information for ipv4 user requests, passing it to subsequent plugins. Requests from ipv6 users will skip this plugin and be processed by later plugins. 2. ipv6: (To be implemented in the future) Only queries geographical location information for ipv6 users, passing it to subsequent plugins. Requests from ipv4 users will skip this plugin and be processed by later plugins. (Currently skips the plugin; requests are handled by subsequent plugins.) |
|  ip_source_type |  string      |  No         |   origin-source    |  Optional values: 1. Peer socket IP: `origin-source`; 2. Retrieved via header: `header`  |
|  ip_header_name |  string      |  No         |   x-forwarded-for  |  When `ip_source_type` is `header`, specify the custom IP source header.                      |

## Configuration Example
```yaml
ip_protocol: ipv4
ip_source_type: header
ip_header_name: X-Real-Ip
``` 

## Explanation for Generating geoCidr.txt
The ip.merge.txt file included in the generateCidr directory is the global IP segment database from the ip2region project on GitHub. The ipRange2Cidr.go program converts IP segments into multiple CIDRs. The converted CIDRs and geographical location information are stored in the /data/geoCidr.txt file. The geo-ip plugin will read the geoCidr.txt file during the configuration stage when Higress starts and parse it into the radixtree data structure in memory for future queries of geographical location information corresponding to user IP addresses. The command to run the conversion program is as follows:
```bash
go run generateCidr/ipRange2Cidr.go
``` 

## Usage of Properties
In the geo-ip plugin, call proxywasm.SetProperty() to set country, city, province, and isp into request attributes so that subsequent plugins can use proxywasm.GetProperty() to obtain the geographical information corresponding to the user's IP for that request.
