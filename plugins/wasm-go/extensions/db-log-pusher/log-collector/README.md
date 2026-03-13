# DB Log Collector Service

This is a tiny log collector service that receives logs from the DB Log Collector plugin and stores them in a MySQL database.

## Overview

The DB Log Collector service is a lightweight HTTP server that receives log entries from the Wasm plugin and stores them in a MySQL database. It supports batch insertion for performance and provides querying capabilities.

## Features

- Receives log entries via HTTP POST at `/ingest`
- Stores logs in MySQL database with 37 different fields
- Supports batch insertion for improved performance
- Provides querying capabilities via `/query` endpoint
- Includes health check endpoint at `/health`

## Configuration

The service can be configured using the following environment variables:

- `MYSQL_DSN`: MySQL Data Source Name (default: root:root@tcp(127.0.0.1:3306)/higress_poc?charset=utf8mb4&parseTime=True)
- `PORT`: Port to listen on (default: 8080)

## Endpoints

- `POST /ingest`: Receive log entries from the plugin
- `GET /query`: Query stored logs with various filters
- `GET /health`: Health check endpoint

## Database Schema

The service expects a table named `access_logs` with 37 fields to store the log entries. The fields include basic request information, response details, traffic metrics, upstream information, connection details, routing information, AI logs, monitoring metadata, and token usage statistics.