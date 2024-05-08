FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY NGS-Karthick-karthick.creds ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o menderdevicesconsumer .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/menderdevicesconsumer .

COPY --from=builder /app/NGS-Karthick-karthick.creds .

CMD ["./menderdevicesconsumer"]
