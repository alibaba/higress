# Copyright (c) 2022 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@proxy_wasm_cpp_sdk//bazel:defs.bzl", "proxy_wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

proxy_wasm_cc_binary(
    name = "basic_auth.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
        "//common:base64.h",
    ],
    deps = [
        "//common:rule_util",
        "//common:json_util",
        "//common:crypto_util",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/time",
    ],
)

cc_library(
    name = "basic_auth_lib",
    srcs = [
        "plugin.cc",
        "//common:base64.h",
    ],
    hdrs = [
        "plugin.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    visibility = ["//visibility:public"],
    alwayslink = 1,
    deps = [
        "//common:rule_util_nullvm",
        "//common:json_util",
        "//common:crypto_util",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/time",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

cc_test(
    name = "basic_auth_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":basic_auth_lib",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
    linkopts = ["-lcrypt"],
)

declare_wasm_image_targets(
    name = "basic_auth",
    wasm_file = ":basic_auth.wasm",
)
