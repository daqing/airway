FROM alpine

WORKDIR /app

RUN mkdir /app/bin
RUN mkdir /app/core
RUN mkdir /app/public

COPY ./bin/airway /app/bin/
COPY ./core /app/core
COPY ./public /app/public

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV AIRWAY_PWD=/app

ENV AW_ASSET_VERSION=1
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/bin/airway"]
