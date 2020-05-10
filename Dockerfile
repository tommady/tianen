FROM golang:latest AS builder

COPY . /app
WORKDIR /app

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o main .

FROM busybox:latest AS runtime
COPY --from=builder /app/main ./

ENTRYPOINT [ "./main" ]
EXPOSE 8080
