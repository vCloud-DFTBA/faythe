FROM golang:1.14.4-alpine as builder
LABEL maintainer="Kien Nguyen-Tuan <kiennt2609@gmail.com>"
ENV GO111MODULE=on
ENV APPLOC=$GOPATH/src/faythe
RUN apk add --no-cache git make bash
ADD . $APPLOC
WORKDIR $APPLOC
RUN make build && \
    chmod +x dist/faythe

FROM alpine:3.12
LABEL maintainer="Kien Nguyen <kiennt2609@gmail.com>"
COPY --from=builder /tmp/faythe /bin/faythe
RUN mkdir -p etc/faythe
COPY examples/faythe.yml /etc/faythe/config.yml
RUN chown -R nobody:nogroup etc/faythe
USER nobody
EXPOSE 8600
ENTRYPOINT ["/bin/faythe"]
CMD ["--config.file", "/etc/faythe/config.yml"]
