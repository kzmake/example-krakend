FROM golang:1.14-alpine AS depbuidler

RUN set -ex \
  && apk add --no-cache -q --no-progress git curl g++ gcc libgcc linux-headers make

WORKDIR /go/app

COPY go.mod .
COPY go.sum .

RUN go mod download


FROM depbuidler as builder

WORKDIR /go/app

COPY . .

RUN set -ex \
  && make build


FROM alpine as release-alpine

WORKDIR /app

COPY conf.json /etc/krakend/conf.json
COPY --from=builder /go/app/target/example-krakend .

RUN addgroup go \
  && adduser -D -G go go \
  && chown -R go:go /app/example-krakend

RUN mkdir -p /app/dump

ENV PATH $PATH:/app

CMD ["example-krakend", "-o", "/app/dump", "-p", "80", "-c", "/etc/krakend/conf.json"]
