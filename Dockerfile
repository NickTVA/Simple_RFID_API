FROM golang:1.24 AS base

WORKDIR /build

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

# Copies everything from your root directory into /app
COPY . .
RUN go build -v -o go-rfid-app

CMD ["/build/go-rfid-app"]


