FROM alpine

WORKDIR /app

RUN mkdir /app/bin

COPY ./bin/airway /app/bin

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV AIRWAY_ROOT=/app
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/bin/airway"]
