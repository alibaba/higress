# Kind cluster creation options

You can control which node image Kind pulls in two ways:

- KIND_NODE_TAG (legacy):
  - Example: `KIND_NODE_TAG=v1.25.3 make create-cluster`
  - Resolves to `kindest/node:${KIND_NODE_TAG}`.

- KIND_NODE_IMAGE (preferred):
  - Example: `KIND_NODE_IMAGE=docker.m.daocloud.io/kindest/node:v1.25.3 make create-cluster`
  - Uses the full image reference. This is recommended when DockerHub is not reachable.

To verify parameter parsing without pulling the image, you can set:

```bash
ONLY_PRINT_NODE_IMAGE=1 tools/hack/create-cluster.sh
```

This prints the resolved `NODE_IMAGE` and exits.

