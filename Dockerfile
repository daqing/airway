FROM alpine

WORKDIR /app

COPY . .

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV AIRWAY_PWD=/app

EXPOSE 1900

CMD ["/app/bin/airway"]
