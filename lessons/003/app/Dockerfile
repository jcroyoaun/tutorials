FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 go build -o resource-hog main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/resource-hog /resource-hog
EXPOSE 8080
ENTRYPOINT ["/resource-hog"]
