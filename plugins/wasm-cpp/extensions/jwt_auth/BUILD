load("@proxy_wasm_cpp_sdk//bazel/wasm:wasm.bzl", "wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

wasm_cc_binary(
    name = "jwt_auth.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
        "extractor.cc",
        "extractor.h",
        "//common:base64.h",
    ],
    deps = [
        "@com_github_google_jwt_verify//:jwt_verify_lib",
        "@com_google_absl//absl/container:btree",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_absl//absl/time",
        "//common:json_util",
        "@proxy_wasm_cpp_sdk//:proxy_wasm_intrinsics",
        "//common:http_util",
        "//common:rule_util",
    ],
)

cc_library(
    name = "jwt_auth_lib",
    srcs = [
        "plugin.cc",
        "extractor.cc",
        "//common:base64.h",
    ],
    hdrs = [
        "plugin.h",
        "extractor.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        "@com_github_google_jwt_verify//:jwt_verify_lib",
        "@com_google_absl//absl/container:btree",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_absl//absl/time",
        "//common:json_util",
        "@proxy_wasm_cpp_host//:lib",
        "//common:http_util_nullvm",
        "//common:rule_util_nullvm",
    ],
)

cc_test(
    name = "jwt_auth_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":jwt_auth_lib",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

declare_wasm_image_targets(
    name = "jwt_auth",
    wasm_file = ":jwt_auth.wasm",
)
