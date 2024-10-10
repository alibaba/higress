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

#![allow(dead_code)]

use proxy_wasm::hostcalls;
use proxy_wasm::types::{BufferType, Bytes, LogLevel, MapType, Status};
use std::time::{Duration, SystemTime};

pub fn get_current_time() -> SystemTime {
    hostcalls::get_current_time().unwrap()
}

pub fn get_property(path: Vec<&str>) -> Option<Bytes> {
    hostcalls::get_property(path).unwrap()
}

pub fn set_property(path: Vec<&str>, value: Option<&[u8]>) {
    hostcalls::set_property(path, value).unwrap()
}

pub fn get_shared_data(key: &str) -> (Option<Bytes>, Option<u32>) {
    hostcalls::get_shared_data(key).unwrap()
}

pub fn set_shared_data(
    key: &str,
    value: Option<&[u8]>,
    cas: Option<u32>,
) -> Result<(), Status> {
    hostcalls::set_shared_data(key, value, cas)
}

pub fn register_shared_queue(name: &str) -> u32 {
    hostcalls::register_shared_queue(name).unwrap()
}

pub fn resolve_shared_queue(vm_id: &str, name: &str) -> Option<u32> {
    hostcalls::resolve_shared_queue(vm_id, name).unwrap()
}

pub fn dequeue_shared_queue(queue_id: u32) -> Result<Option<Bytes>, Status> {
    hostcalls::dequeue_shared_queue(queue_id)
}

pub fn enqueue_shared_queue(queue_id: u32, value: Option<&[u8]>) -> Result<(), Status> {
    hostcalls::enqueue_shared_queue(queue_id, value)
}

pub fn dispatch_http_call(
    upstream: &str,
    headers: Vec<(&str, &str)>,
    body: Option<&[u8]>,
    trailers: Vec<(&str, &str)>,
    timeout: Duration,
) -> Result<u32, Status> {
    hostcalls::dispatch_http_call(upstream, headers, body, trailers, timeout)
}

pub fn get_http_call_response_headers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpCallResponseHeaders).unwrap()
}

pub fn get_http_call_response_headers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpCallResponseHeaders).unwrap()
}

pub fn get_http_call_response_header(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpCallResponseHeaders, name).unwrap()
}

pub fn get_http_call_response_header_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpCallResponseHeaders, name).unwrap()
}

pub fn get_http_call_response_body(start: usize, max_size: usize) -> Option<Bytes> {
    hostcalls::get_buffer(BufferType::HttpCallResponseBody, start, max_size).unwrap()
}

pub fn get_http_call_response_trailers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpCallResponseTrailers).unwrap()
}

pub fn get_http_call_response_trailers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpCallResponseTrailers).unwrap()
}

pub fn get_http_call_response_trailer(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpCallResponseTrailers, name).unwrap()
}

pub fn get_http_call_response_trailer_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpCallResponseTrailers, name).unwrap()
}

pub fn dispatch_grpc_call(
    upstream_name: &str,
    service_name: &str,
    method_name: &str,
    initial_metadata: Vec<(&str, &[u8])>,
    message: Option<&[u8]>,
    timeout: Duration,
) -> Result<u32, Status> {
    hostcalls::dispatch_grpc_call(
        upstream_name,
        service_name,
        method_name,
        initial_metadata,
        message,
        timeout,
    )
}

pub fn get_grpc_call_response_body(start: usize, max_size: usize) -> Option<Bytes> {
    hostcalls::get_buffer(BufferType::GrpcReceiveBuffer, start, max_size).unwrap()
}

pub fn cancel_grpc_call(token_id: u32) {
    hostcalls::cancel_grpc_call(token_id).unwrap()
}

pub fn open_grpc_stream(
    cluster_name: &str,
    service_name: &str,
    method_name: &str,
    initial_metadata: Vec<(&str, &[u8])>,
) -> Result<u32, Status> {
    hostcalls::open_grpc_stream(cluster_name, service_name, method_name, initial_metadata)
}

pub fn get_grpc_stream_initial_metadata() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::GrpcReceiveInitialMetadata).unwrap()
}

pub fn get_grpc_stream_initial_metadata_value(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::GrpcReceiveInitialMetadata, name).unwrap()
}

pub fn send_grpc_stream_message(token_id: u32, message: Option<&[u8]>, end_stream: bool) {
    hostcalls::send_grpc_stream_message(token_id, message, end_stream).unwrap()
}

pub fn get_grpc_stream_trailing_metadata() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::GrpcReceiveTrailingMetadata).unwrap()
}

pub fn get_grpc_stream_trailing_metadata_value(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::GrpcReceiveTrailingMetadata, name).unwrap()
}

pub fn cancel_grpc_stream(token_id: u32) {
    hostcalls::cancel_grpc_stream(token_id).unwrap()
}

pub fn close_grpc_stream(token_id: u32) {
    hostcalls::close_grpc_stream(token_id).unwrap()
}

pub fn get_grpc_status() -> (u32, Option<String>) {
    hostcalls::get_grpc_status().unwrap()
}

pub fn call_foreign_function(
    function_name: &str,
    arguments: Option<&[u8]>,
) -> Result<Option<Bytes>, Status> {
    hostcalls::call_foreign_function(function_name, arguments)
}

pub fn done() {
    hostcalls::done().unwrap()
}

pub fn get_http_request_headers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpRequestHeaders).unwrap()
}

pub fn get_http_request_headers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpRequestHeaders).unwrap()
}

pub fn set_http_request_headers(headers: Vec<(&str, &str)>) {
    hostcalls::set_map(MapType::HttpRequestHeaders, headers).unwrap()
}

pub fn set_http_request_headers_bytes(headers: Vec<(&str, &[u8])>) {
    hostcalls::set_map_bytes(MapType::HttpRequestHeaders, headers).unwrap()
}

pub fn get_http_request_header(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpRequestHeaders, name).unwrap()
}

pub fn get_http_request_header_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpRequestHeaders, name).unwrap()
}

pub fn set_http_request_header(name: &str, value: Option<&str>) {
    hostcalls::set_map_value(MapType::HttpRequestHeaders, name, value).unwrap()
}

pub fn set_http_request_header_bytes(name: &str, value: Option<&[u8]>) {
    hostcalls::set_map_value_bytes(MapType::HttpRequestHeaders, name, value).unwrap()
}

pub fn add_http_request_header(name: &str, value: &str) {
    hostcalls::add_map_value(MapType::HttpRequestHeaders, name, value).unwrap()
}

pub fn add_http_request_header_bytes(name: &str, value: &[u8]) {
    hostcalls::add_map_value_bytes(MapType::HttpRequestHeaders, name, value).unwrap()
}

pub fn get_http_request_body(start: usize, max_size: usize) -> Option<Bytes> {
    hostcalls::get_buffer(BufferType::HttpRequestBody, start, max_size).unwrap()
}

pub fn set_http_request_body(start: usize, size: usize, value: &[u8]) {
    hostcalls::set_buffer(BufferType::HttpRequestBody, start, size, value).unwrap()
}

pub fn get_http_request_trailers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpRequestTrailers).unwrap()
}

pub fn get_http_request_trailers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpRequestTrailers).unwrap()
}

pub fn set_http_request_trailers(trailers: Vec<(&str, &str)>) {
    hostcalls::set_map(MapType::HttpRequestTrailers, trailers).unwrap()
}

pub fn set_http_request_trailers_bytes(trailers: Vec<(&str, &[u8])>) {
    hostcalls::set_map_bytes(MapType::HttpRequestTrailers, trailers).unwrap()
}

pub fn get_http_request_trailer(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpRequestTrailers, name).unwrap()
}

pub fn get_http_request_trailer_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpRequestTrailers, name).unwrap()
}

pub fn set_http_request_trailer(name: &str, value: Option<&str>) {
    hostcalls::set_map_value(MapType::HttpRequestTrailers, name, value).unwrap()
}

pub fn set_http_request_trailer_bytes(name: &str, value: Option<&[u8]>) {
    hostcalls::set_map_value_bytes(MapType::HttpRequestTrailers, name, value).unwrap()
}

pub fn add_http_request_trailer(name: &str, value: &str) {
    hostcalls::add_map_value(MapType::HttpRequestTrailers, name, value).unwrap()
}

pub fn add_http_request_trailer_bytes(name: &str, value: &[u8]) {
    hostcalls::add_map_value_bytes(MapType::HttpRequestTrailers, name, value).unwrap()
}

pub fn resume_http_request() {
    hostcalls::resume_http_request().unwrap()
}

pub fn reset_http_request() {
    hostcalls::reset_http_request().unwrap()
}

pub fn get_http_response_headers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpResponseHeaders).unwrap()
}

pub fn get_http_response_headers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpResponseHeaders).unwrap()
}

pub fn set_http_response_headers(headers: Vec<(&str, &str)>) {
    hostcalls::set_map(MapType::HttpResponseHeaders, headers).unwrap()
}

pub fn set_http_response_headers_bytes(headers: Vec<(&str, &[u8])>) {
    hostcalls::set_map_bytes(MapType::HttpResponseHeaders, headers).unwrap()
}

pub fn get_http_response_header(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpResponseHeaders, name).unwrap()
}

pub fn get_http_response_header_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpResponseHeaders, name).unwrap()
}

pub fn set_http_response_header(name: &str, value: Option<&str>) {
    hostcalls::set_map_value(MapType::HttpResponseHeaders, name, value).unwrap()
}

pub fn set_http_response_header_bytes(name: &str, value: Option<&[u8]>) {
    hostcalls::set_map_value_bytes(MapType::HttpResponseHeaders, name, value).unwrap()
}

pub fn add_http_response_header(name: &str, value: &str) {
    hostcalls::add_map_value(MapType::HttpResponseHeaders, name, value).unwrap()
}

pub fn add_http_response_header_bytes(name: &str, value: &[u8]) {
    hostcalls::add_map_value_bytes(MapType::HttpResponseHeaders, name, value).unwrap()
}

pub fn get_http_response_body(start: usize, max_size: usize) -> Option<Bytes> {
    hostcalls::get_buffer(BufferType::HttpResponseBody, start, max_size).unwrap()
}

pub fn set_http_response_body(start: usize, size: usize, value: &[u8]) {
    hostcalls::set_buffer(BufferType::HttpResponseBody, start, size, value).unwrap()
}

pub fn get_http_response_trailers() -> Vec<(String, String)> {
    hostcalls::get_map(MapType::HttpResponseTrailers).unwrap()
}

pub fn get_http_response_trailers_bytes() -> Vec<(String, Bytes)> {
    hostcalls::get_map_bytes(MapType::HttpResponseTrailers).unwrap()
}

pub fn set_http_response_trailers(trailers: Vec<(&str, &str)>) {
    hostcalls::set_map(MapType::HttpResponseTrailers, trailers).unwrap()
}

pub fn set_http_response_trailers_bytes(trailers: Vec<(&str, &[u8])>) {
    hostcalls::set_map_bytes(MapType::HttpResponseTrailers, trailers).unwrap()
}

pub fn get_http_response_trailer(name: &str) -> Option<String> {
    hostcalls::get_map_value(MapType::HttpResponseTrailers, name).unwrap()
}

pub fn get_http_response_trailer_bytes(name: &str) -> Option<Bytes> {
    hostcalls::get_map_value_bytes(MapType::HttpResponseTrailers, name).unwrap()
}

pub fn set_http_response_trailer(name: &str, value: Option<&str>) {
    hostcalls::set_map_value(MapType::HttpResponseTrailers, name, value).unwrap()
}

pub fn set_http_response_trailer_bytes(name: &str, value: Option<&[u8]>) {
    hostcalls::set_map_value_bytes(MapType::HttpResponseTrailers, name, value).unwrap()
}

pub fn add_http_response_trailer(name: &str, value: &str) {
    hostcalls::add_map_value(MapType::HttpResponseTrailers, name, value).unwrap()
}

pub fn add_http_response_trailer_bytes(name: &str, value: &[u8]) {
    hostcalls::add_map_value_bytes(MapType::HttpResponseTrailers, name, value).unwrap()
}

pub fn resume_http_response() {
    hostcalls::resume_http_response().unwrap()
}

pub fn reset_http_response() {
    hostcalls::reset_http_response().unwrap()
}

pub fn send_http_response(
    status_code: u32,
    headers: Vec<(&str, &str)>,
    body: Option<&[u8]>,
) {
    hostcalls::send_http_response(status_code, headers, body).unwrap()
}

pub fn log(level: LogLevel, message: &str) {
    hostcalls::log(level, message).unwrap()
}