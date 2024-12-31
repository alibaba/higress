cd /Users/daijingze/GolandProjects/github/higress/plugins/wasm-go
PLUGIN_NAME=oidc make build
cd extensions/oidc/
docker build -t oidc-wasm .
image_id=$(docker images | grep "^oidc-wasm " | awk '{print $3}')
docker tag $image_id registry.cn-hangzhou.aliyuncs.com/jingze/oidc:1.0.0
docker push registry.cn-hangzhou.aliyuncs.com/jingze/oidc:1.0.0