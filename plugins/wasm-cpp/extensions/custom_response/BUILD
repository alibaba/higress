load("@proxy_wasm_cpp_sdk//bazel/wasm:wasm.bzl", "wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

wasm_cc_binary(
    name = "custom_response.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
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
    name = "custom_response_lib",
    srcs = [
        "plugin.cc",
    ],
    hdrs = [
        "plugin.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        "@com_google_absl//absl/strings",
        "//common:json_util",
        "@proxy_wasm_cpp_host//:lib",
        "//common:http_util_nullvm",
        "//common:rule_util_nullvm",
    ],
)

cc_test(
    name = "custom_response_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":custom_response_lib",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

declare_wasm_image_targets(
    name = "custom_response",
    wasm_file = ":custom_response.wasm",
)
