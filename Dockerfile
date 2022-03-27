FROM golang:1.18-alpine as build
COPY . /build

RUN apk add \
    git \
&&  cd /build \
&&  go build -ldflags="-s -w"

FROM alpine
COPY --from=build /build/go-discord-amputator /

CMD ["/go-discord-amputator"]
