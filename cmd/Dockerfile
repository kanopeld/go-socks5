FROM golang

ENV PROXY_PORT=5300
ENV LOG_LEVEL="error"

ADD . /go/src/github.com/kanopeld/go-socks5/cmd
WORKDIR /go/src/github.com/kanopeld/go-socks5/cmd
RUN go mod download
RUN go install github.com/kanopeld/go-socks5/cmd
RUN chmod +x /go/src/github.com/kanopeld/go-socks5/cmd/docker-entrypoint.sh

ENTRYPOINT ["/go/src/github.com/kanopeld/go-socks5/cmd/docker-entrypoint.sh"]

EXPOSE $PROXY_PORT