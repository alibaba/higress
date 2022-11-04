load("@proxy_wasm_cpp_sdk//bazel/wasm:wasm.bzl", "wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

wasm_cc_binary(
    name = "key_rate_limit.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
        "bucket.h",
        "bucket.cc",
    ],
    deps = [
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/time",
        "//common:json_util",
        "@proxy_wasm_cpp_sdk//:proxy_wasm_intrinsics",
        "//common:http_util",
        "//common:rule_util",
    ],
)

cc_library(
    name = "key_rate_limit_lib",
    srcs = [
        "plugin.cc",
        "bucket.cc",
    ],
    hdrs = [
        "plugin.h",
        "bucket.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        "@com_google_absl//absl/strings",
        "//common:json_util",
        "@proxy_wasm_cpp_host//:lib",
        "//common:http_util",
        "//common:rule_util",
    ],
)

cc_test(
    name = "key_rate_limit_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":key_rate_limit_lib",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

declare_wasm_image_targets(
    name = "key_rate_limit",
    wasm_file = ":key_rate_limit.wasm",
)
