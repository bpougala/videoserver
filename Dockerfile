FROM golang:1.13 as builder

WORKDIR /app
RUN go mod init videoServer.go
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server
FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /server
CMD ["/server"]
