FROM golang

ENV PROXY_PORT=5300
ENV LOG_LEVEL="error"

ADD . /go/src/github.com/kanopeld/go-socks5/cmd
WORKDIR /go/src/github.com/kanopeld/go-socks5/cmd
RUN go mod download
RUN go install github.com/kanopeld/go-socks5/cmd

ENTRYPOINT ["PROXY_PORT=$PROXY_PORT LOG_LEVEL=$LOG_LEVEL /go/bin/cmd"]

EXPOSE $PROXY_PORT