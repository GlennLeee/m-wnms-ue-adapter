#!/bin/sh

# 로그 디렉토리/파일
LOG_DIR="/app/log"
LOG_FILE="${LOG_DIR}/wnms-ue-adapter.log"

# 애플리케이션 실행 및 로그 기록
exec /app/wnms-ue-adapter 2>&1 | tee -a "$LOG_FILE"