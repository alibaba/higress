load("@proxy_wasm_cpp_sdk//bazel/wasm:wasm.bzl", "wasm_cc_binary")
load("//bazel:wasm.bzl", "declare_wasm_image_targets")

wasm_cc_binary(
    name = "sni_misdirect.wasm",
    srcs = [
        "plugin.cc",
        "plugin.h",
    ],
    deps = [
        "@com_google_absl//absl/strings:str_format",        
        "@com_google_absl//absl/strings",
        "@proxy_wasm_cpp_sdk//:proxy_wasm_intrinsics",
        "//common:http_util",
    ],
)

cc_library(
    name = "sni_misdirect_lib",
    srcs = [
        "plugin.cc",
    ],
    hdrs = [
        "plugin.h",
    ],
    copts = ["-DNULL_PLUGIN"],
    visibility = ["//visibility:public"],
    alwayslink = 1,
    deps = [
        "@com_google_absl//absl/strings:str_format",
        "@com_google_absl//absl/strings",
        "@proxy_wasm_cpp_host//:lib",
        "//common:http_util",
    ],
)

cc_test(
    name = "sni_misdirect_test",
    srcs = [
        "plugin_test.cc",
    ],
    copts = ["-DNULL_PLUGIN"],
    deps = [
        ":sni_misdirect_lib",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_googletest//:gtest",
        "@com_google_googletest//:gtest_main",
        "@proxy_wasm_cpp_host//:lib",
    ],
)

declare_wasm_image_targets(
    name = "sni_misdirect",
    wasm_file = ":sni_misdirect.wasm",
)
