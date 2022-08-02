FROM golang:1.18-alpine as builder
LABEL maintainer="Tuan Dat Vu <tuandatk25a@gmail.com>"
ENV GO111MODULE=on
ENV APPLOC=$GOPATH/src/faythe
RUN apk add --no-cache git make bash
ADD . $APPLOC
WORKDIR $APPLOC
RUN GO_OUT=/bin make build && \
    chmod +x /bin/faythe

FROM alpine:3.12
LABEL maintainer="Tuan Dat Vu <tuandatk25a@gmail.com>"
COPY --from=builder /bin/faythe /bin/faythe
RUN mkdir -p etc/faythe
COPY examples/faythe.yml /etc/faythe/config.yml
RUN chown -R nobody:nogroup etc/faythe
USER nobody
EXPOSE 8600
ENTRYPOINT ["/bin/faythe"]
CMD ["--config.file", "/etc/faythe/config.yml"]
