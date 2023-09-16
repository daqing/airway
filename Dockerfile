FROM golang:1.21.1-bookworm

WORKDIR /app

COPY ./bin/airway /app/

ENV AIRWAY_ENV=production
ENV PORT=1900

EXPOSE 1900

CMD ["/app/airway"]
