FROM alpine:3.8

RUN apk add --no-cache bash make ca-certificates && update-ca-certificates

COPY bin/duffle /usr/bin/duffle

CMD /usr/bin/duffle