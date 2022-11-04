#!/bin/bash

read -p "please enter the env(prod,pre): " env

repo=""

case $env in
    prod)
        repo="platform_wasm"
        echo "注意！正在操作生产环境"
        ;;
    pre)
        repo="platform_wasm_pre"
        ;;
    *)
        echo "unknown env: "$env
        exit
esac

read -p "please enter the registry addr: " registry_addr
read -p "please enter username: " username
read -p "please enter password: " -s password


plugins=("basic-auth" "bot-detect" "custom-response" "hmac-auth" "key-auth" "key-rate-limit" "request-block" "sni-misdirect" "jwt-auth")

for plugin in ${plugins[@]}; do
    dir_name=`echo $plugin | tr '-' '_'`
    bazel build //extensions/$dir_name:$dir_name.wasm
    oras push -u $username -p $password $registry_addr/$repo/$plugin:1.0.0 \
         config.json:application/vnd.module.wasm.config.v1+json  \
         bazel-bin/extensions/$dir_name/$dir_name.wasm:application/vnd.module.wasm.content.layer.v1+wasm
done
