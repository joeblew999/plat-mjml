FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 1

RUN apk update --no-cache && apk add --no-cache tzdata gcc musl-dev

WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -o /app/server ./cmd/server


FROM alpine

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/UTC /usr/share/zoneinfo/UTC
ENV TZ UTC

WORKDIR /app
COPY --from=builder /app/server /app/server
COPY --from=builder /build/templates /app/templates
COPY --from=builder /build/config.yaml /app/config.yaml

EXPOSE 8080 8081 8082

CMD ["./server"]
