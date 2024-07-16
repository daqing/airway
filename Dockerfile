FROM alpine

WORKDIR /app

RUN mkdir /app/bin
RUN mkdir /app/core
RUN mkdir /app/ext
RUN mkdir /app/views
RUN mkdir /app/public

COPY ./bin/airway /app/bin
COPY ./bin/cli_amd /app/bin
COPY ./core /app/core
COPY ./ext /app/ext
COPY ./views /app/views
COPY ./public /app/public

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900

ENV AIRWAY_ROOT=/app
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/bin/airway"]
