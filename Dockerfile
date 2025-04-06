# modules cache
FROM golang:1.24-alpine as modules

COPY go.mod go.sum Makefile /modules/
WORKDIR /modules

RUN go mod download

# building application
FROM golang:1.24-alpine as builder

RUN apk add musl-dev make gcc libc-dev curl git

WORKDIR /app

COPY --from=modules /go/pkg /go/pkg
COPY . /app

RUN CGO_ENABLED=0 go build -o /bin/app .

# application image
FROM alpine

COPY --from=builder /bin/app /usr/bin/app

CMD ["sh", "-c", "/usr/bin/app"]