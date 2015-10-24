FROM golang

ADD . /go/src/github.com/macb/notif
RUN go get github.com/macb/notif/...

RUN go install github.com/macb/notif

ENTRYPOINT /go/bin/notif
