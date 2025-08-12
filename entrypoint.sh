#!/bin/bash

# 로그 파일 경로
LOG_FILE="/app/log/awnms-p5g-adapter.log"

# 애플리케이션 실행 및 로그 기록
/app/app -c /app/conf.d/properties.ini >> "$LOG_FILE" 2>&1