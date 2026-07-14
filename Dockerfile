FROM debian:bookworm-slim

LABEL AUTHOR="Glenn Lee (glenn.lee@connectlabs.co.kr)"

# 필요한 최소 패키지만 설치 (타임존/인증서)
RUN apt-get update \
    && apt-get install -y --no-install-recommends tzdata ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app/conf.d /app/log

COPY wnms-ue-adapter /app/wnms-ue-adapter
COPY entrypoint.sh /entrypoint.sh

RUN chmod a+x /entrypoint.sh \
    && chmod a+x /app/wnms-ue-adapter

# entrypoint 실행
ENTRYPOINT ["/entrypoint.sh"]