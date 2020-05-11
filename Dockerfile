FROM golang:latest AS builder

COPY . /app
WORKDIR /app

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o main .

FROM alpine as certs
RUN apk update && apk add ca-certificates

FROM busybox:latest AS runtime
COPY --from=builder /app/main ./
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

ENTRYPOINT [ "./main" ]
EXPOSE 8080
