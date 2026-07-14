docker run --rm \
  --platform linux/amd64 \
  -e SONAR_HOST_URL="http://10.153.30.162:9000" \
  -e SONAR_TOKEN="sqp_0440b4880975ff2b610b25b528a6293982a0ef2b" \
  -v "$(pwd):/usr/src" \
  sonarsource/sonar-scanner-cli \
  -Dsonar.projectKey=mwnms-ue-adapter \
  -Dsonar.sources=. \
  -Dsonar.scm.disabled=true \
  -Dsonar.exclusions="test/**,**/test/**,**/*_test.go,Dockerfile,**/Dockerfile,.gocache/**"