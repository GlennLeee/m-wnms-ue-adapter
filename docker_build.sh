#!/bin/bash

if [ $# -ne 1 ]; then
	echo "버전 없음, docker images 를 활용하여 버전을 적어주세요."
    exit 1
fi

echo "빌드 버전: $1"

echo "BUILD GOLANG SOURCE"
GOOS=linux GOARCH=amd64 go build --tags gosnmp_nodebug

echo "BUILD DOCKER IMAGE"
docker build --platform linux/amd64 -t 10.153.30.100/awnms/awnms-p5g-adapter:latest .
docker build --platform linux/amd64 -t 10.153.30.100/awnms/awnms-p5g-adapter:$1 .
#docker build --platform linux/amd64 -t 192.168.1.122:8111/awnms/awnms-p5g-adapter:latest .

echo "PUSH DOCKER IMAGE"
docker push 10.153.30.100/awnms/awnms-p5g-adapter:latest
docker push 10.153.30.100/awnms/awnms-p5g-adapter:$1
#docker push 192.168.1.122:8111/awnms/awnms-p5g-adapter:latest