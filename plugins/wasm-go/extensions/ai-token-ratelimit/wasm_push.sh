cat <<EOF > Dockerfile 
FROM scratch

COPY main.wasm plugin.wasm
EOF

image_name=$(basename $(pwd))
echo $image_name
# personal
# docker build -t registry.cn-hangzhou.aliyuncs.com/rinfx/$image_name:1.0.0 .
# docker push registry.cn-hangzhou.aliyuncs.com/rinfx/$image_name:1.0.0

## mse prod
docker build -t msecrinstance-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0 .
docker push msecrinstance-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0

## higress
# docker build -t higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/$image_name:1.0.0 .
# docker push higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/$image_name:1.0.0

## mse finance
docker build -t msecrinstance-registry.cn-shanghai-finance-1.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0 .
docker push msecrinstance-registry.cn-shanghai-finance-1.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0

## apig pre
docker build -t wasm-manager-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm_pre/$image_name:1.0.0 .
docker push wasm-manager-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm_pre/$image_name:1.0.0

## apig prod
docker build -t wasm-manager-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0 .
docker push wasm-manager-registry.cn-hangzhou.cr.aliyuncs.com/platform_wasm/$image_name:1.0.0

rm Dockerfile