FROM golang:1.26.5-alpine3.24 AS builder
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tpp-app ./cmd/server

FROM alpine:3.24.1
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/tpp-app /app/tpp-app
USER app
ENV PORT=8080 TZ=Asia/Jakarta
EXPOSE 8080
ENTRYPOINT ["/app/tpp-app"]
