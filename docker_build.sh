#!/bin/bash

echo "BUILD GOLANG SOURCE"
GOOS=linux GOARCH=amd64 go build

echo "BUILD DOCKER IMAGE"
docker build --platform linux/amd64 -t docker.remonn:8111/wnms/wnms-ue-adapter:latest .

echo "PUSH DOCKER IMAGE"
docker push docker.remonn:8111/wnms/wnms-ue-adapter:latest