load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file")
load("@bazel_skylib//rules:copy_file.bzl", "copy_file")
load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_image",
    "container_push",
)

def wasm_libraries():
    http_archive(
        name = "com_google_absl",
        sha256 = "ec8ef47335310cc3382bdc0d0cc1097a001e67dc83fcba807845aa5696e7e1e4",
        strip_prefix = "abseil-cpp-302b250e1d917ede77b5ff00a6fd9f28430f1563",
        url = "https://github.com/abseil/abseil-cpp/archive/302b250e1d917ede77b5ff00a6fd9f28430f1563.tar.gz",
        patch_args = ["-p1"],
        patches = ["//bazel:absl.patch"],
    )

    http_file(
        name = "com_github_nlohmann_json_single_header",
        sha256 = "3b5d2b8f8282b80557091514d8ab97e27f9574336c804ee666fda673a9b59926",
        urls = [
            "https://github.com/nlohmann/json/releases/download/v3.7.3/json.hpp",
        ],
    )


    # import google test and cpp host for unit testing
    http_archive(
        name = "com_google_googletest",
        sha256 = "9dc9157a9a1551ec7a7e43daea9a694a0bb5fb8bec81235d8a1e6ef64c716dcb",
        strip_prefix = "googletest-release-1.10.0",
        urls = ["https://github.com/google/googletest/archive/release-1.10.0.tar.gz"],
    )

    PROXY_WASM_CPP_HOST_SHA = "f38347360feaaf5b2a733f219c4d8c9660d626f0"
    PROXY_WASM_CPP_HOST_SHA256 = "bf10de946eb5785813895c2bf16504afc0cd590b9655d9ee52fb1074d0825ea3"

    http_archive(
        name = "proxy_wasm_cpp_host",
        sha256 = PROXY_WASM_CPP_HOST_SHA256,
        strip_prefix = "proxy-wasm-cpp-host-" + PROXY_WASM_CPP_HOST_SHA,
        url = "https://github.com/proxy-wasm/proxy-wasm-cpp-host/archive/" + PROXY_WASM_CPP_HOST_SHA +".tar.gz",
    )

    http_archive(
        name = "boringssl",
        urls = ["https://github.com/google/boringssl/archive/648cbaf033401b7fe7acdce02f275b06a88aab5c.tar.gz"],
        strip_prefix = "boringssl-648cbaf033401b7fe7acdce02f275b06a88aab5c",
        patch_args = ["-p1"],
        patches = ["//bazel:boringssl.patch"],
    )

    native.bind(
        name = "ssl",
        actual = "@boringssl//:ssl",
    )

    http_archive(
        name = "com_google_protobuf",
        urls = ["https://github.com/protocolbuffers/protobuf/releases/download/v{version}/protobuf-all-3.18.0.tar.gz"],
        strip_prefix = "protobuf-3.18.0",
    )

    native.bind(
        name = "protobuf",
        actual = "@com_google_protobuf//:protobuf",
    )

    http_archive(
        name = "com_googlesource_code_re2",
        urls = ["https://github.com/google/re2/archive/2020-07-06.tar.gz"],
        strip_prefix = "re2-2020-07-06",
        patch_args = ["-p1"],
        patches = ["//bazel:re2.patch"],
    )

    native.bind(
        name = "abseil_flat_hash_set",
        actual = "@com_google_absl//absl/container:flat_hash_set",
    )

    native.bind(
        name = "abseil_strings",
        actual = "@com_google_absl//absl/strings:strings",
    )

    native.bind(
        name = "abseil_time",
        actual = "@com_google_absl//absl/time:time",
    )

    native.bind(
        name = "protobuf",
        actual = "@com_google_protobuf//:protobuf",
    )

    http_archive(
        name = "com_github_google_jwt_verify",
        urls = ["https://github.com/google/jwt_verify_lib/archive/26c22c0ce1bc607eec8fa5dd26b707378adc7a88.tar.gz"],
        strip_prefix = "jwt_verify_lib-26c22c0ce1bc607eec8fa5dd26b707378adc7a88"
    )
    
    
def declare_wasm_image_targets(name, wasm_file):
    # Rename to the spec compatible name.
    copy_file("copy_original_file", wasm_file, "plugin.wasm")
    container_image(
        name = "wasm_image",
        files = [":plugin.wasm"],
    )
    container_push(
        name = "push_wasm_image",
        format = "OCI",
        image = ":wasm_image",
        registry = "ghcr.io",
        repository = "istio-ecosystem/wasm-extensions/"+name,
        tag = "$(WASM_IMAGE_TAG)",
    )
