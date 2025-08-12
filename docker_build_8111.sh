#!/bin/bash

if [ $# -ne 1 ]; then
	echo "버전 없음, docker images 를 활용하여 버전을 적어주세요."
    exit 1
fi

echo "빌드 버전: $1"

echo "BUILD GOLANG SOURCE"
GOOS=linux GOARCH=amd64 go build --tags gosnmp_nodebug

echo "BUILD DOCKER IMAGE"
docker build --platform linux/amd64 -t ${HARBOR}/awnms/awnms-p5g-adapter:latest .
docker build --platform linux/amd64 -t ${HARBOR}/awnms/awnms-p5g-adapter:$1 .

#echo "PUSH DOCKER IMAGE"
#docker push ${HARBOR}/awnms/awnms-p5g-adapter:latest
#docker push ${HARBOR}/awnms/awnms-p5g-adapter:$1
