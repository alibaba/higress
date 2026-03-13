# DB Log Pusher Plugin

This plugin collects HTTP request logs and pushes them to a database log collector service.

## Overview

The DB Log Pusher plugin captures HTTP request/response details and pushes them to a designated collector service for storage and analysis. It includes support for monitoring metadata fields and token usage statistics.

## Features

- Captures comprehensive HTTP request/response information
- Supports monitoring metadata fields (instance ID, API, model, consumer, etc.)
- Includes token usage statistics (input, output, and total tokens)
- Pushes logs asynchronously to collector service
- Configurable collector service endpoint

## Configuration

The plugin accepts the following configuration parameters:

```yaml
config:
  collector_service_name: "db-log-collector"  # Name of the collector service
  collector_port: 80                        # Port of the collector service  
  collector_path: "/ingest"                 # API path for sending logs
```

## Usage

Apply the plugin to your Higress gateway using the WasmPlugin CRD as shown in the example deployment file.

Note: This plugin acts as a "pusher" that sends logs to a collector service. The collector service receives and stores the logs.

## Fields Collected

The plugin captures 37 different fields including:
- Basic request information (time, authority, method, path, etc.)
- Response details (status code, flags, timing)
- Traffic metrics (bytes received/sent, duration)
- Upstream and connection information
- Monitoring metadata (instance ID, API, model, consumer, etc.)
- Token usage statistics