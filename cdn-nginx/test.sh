#!/bin/bash

set -ex

docker build -q --rm --build-arg http_proxy='http://proxy-dmz.intel.com:911' --build-arg https_proxy='http://proxy-dmz.intel.com:912' --build-arg HTTP_PROXY='http://proxy-dmz.intel.com:911' --build-arg HTTPS_PROXY='http://proxy-dmz.intel.com:912' --build-arg NO_PROXY='localhost,*.intel.com,intel.com,127.0.0.1' --build-arg no_proxy='localhost,*.intel.com,intel.com,127.0.0.1' -t amr-registry.caas.intel.com/one-intel-edge/maestro-i/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-nginx:1.0.1-dev -f cdn-nginx/Dockerfile cdn-nginx
docker run --name cdn-nginx --rm -d  -p 8080:8080 --network host --env BOOTS_SERVICE_URL=localhost amr-registry.caas.intel.com/one-intel-edge/maestro-i/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-nginx:1.0.1-dev >/dev/null 2>&1
sleep 10
curl -s "http://localhost:8080/index.php" --noproxy '*' --output result.html
docker stop cdn-nginx
if cmp -s "result.html" "reference.html"; then
    exit 0
else
    exit 1
fi

