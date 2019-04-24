FROM golang:1.12-alpine as builder

ENV GO111MODULE=on
ENV APPLOC=$GOPATH/src/faythe

RUN apk add --no-cache git

ADD . $APPLOC
WORKDIR $APPLOC
RUN go build -mod vendor -o /bin/faythe

FROM golang:1.12-alpine
LABEL maintainer="Kien Nguyen <kiennt2609@gmail.com>"
COPY --from=builder /bin/faythe /bin/faythe
RUN chmod +x /bin/faythe
EXPOSE 8600
ENTRYPOINT ["/bin/faythe"]
