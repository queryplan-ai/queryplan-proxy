# syntax=docker/dockerfile:1
FROM golang:1.21 as builder

EXPOSE 3000

WORKDIR /go/src/github.com/queryplan-ai/queryplan-proxy

COPY . .

RUN make build

CMD ["make", "run"]
