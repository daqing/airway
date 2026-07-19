FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . /app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache gcc musl-dev
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=1 go build -o ./bin/airway .

FROM alpine
WORKDIR /app
COPY --from=builder /app/bin/airway /app
COPY --from=builder /app/db /app/db

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/airway"]
