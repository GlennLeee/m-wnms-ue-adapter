FROM ubuntu:20.04
LABEL AUTHOR Glenn Lee (glenn.lee@biskitlab.com)
RUN apt-get update -y && apt-get upgrade -y
RUN apt-get install -y tzdata

COPY p5g-data-adapter /app/app
COPY entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh
RUN chmod a+x /app/app

# entrypoint 실행
ENTRYPOINT ["/entrypoint.sh"]