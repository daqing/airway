FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . /app
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go build -o ./bin/airway .

FROM alpine
WORKDIR /app
COPY --from=builder /app/bin/airway /app

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/airway"]
