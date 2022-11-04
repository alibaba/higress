load("@proxy_wasm_cpp_sdk//bazel/wasm:wasm.bzl", "wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

wasm_cc_binary(
    name = "key_auth.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
        "//common:base64.h",
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
    name = "key_auth_lib",
    srcs = [
        "plugin.cc",
        "//common:base64.h",
    ],
    hdrs = [
        "plugin.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/time",
        "//common:json_util",
        "@proxy_wasm_cpp_host//:lib",
        "//common:http_util",
        "//common:rule_util",
    ],
)

cc_test(
    name = "key_auth_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":key_auth_lib",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

declare_wasm_image_targets(
    name = "key_auth",
    wasm_file = ":key_auth.wasm",
)
