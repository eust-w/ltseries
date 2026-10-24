FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ltseries-api ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates ffmpeg
WORKDIR /app
COPY --from=build /out/ltseries-api /app/ltseries-api
EXPOSE 18080
ENTRYPOINT ["/app/ltseries-api"]
