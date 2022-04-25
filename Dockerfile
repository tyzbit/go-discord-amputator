FROM golang:1.18-alpine as build
WORKDIR /
COPY . ./

RUN apk add \
    build-base \
    git \
&&  go build -ldflags="-s -w"

FROM alpine
ENV GIN_MODE=release

COPY --from=build /go-discord-amputator /

CMD ["/go-discord-amputator"]
