FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/selectos-auth .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget && addgroup -S selectos && adduser -S -G selectos selectos
WORKDIR /app
COPY --from=build /out/selectos-auth /usr/local/bin/selectos-auth
USER selectos
EXPOSE 8088
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8088/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/selectos-auth"]
