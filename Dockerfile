# This file is intended to be used apart from the containing source code tree.

FROM golang:alpine AS builder
ARG USERNAME=jrarj
ARG PASSWORD=H3110fr13nD
RUN apk add apache2-utils
RUN htpasswd -bBc /tmp/users $USERNAME $PASSWORD
WORKDIR /usr/local/go/src/caldavserver
COPY caldavserver/caldavserver.go .
COPY caldavserver/go.mod .
RUN go get golang.org/x/crypto/bcrypt
RUN go get golang.org/x/net/webdav
RUN go build -o caldavserver .


FROM python:3-alpine
ARG VERSION=v3.1.8
EXPOSE 3000
EXPOSE 4000
RUN apk add --no-cache ca-certificates openssl \
 && apk add --no-cache --virtual .build-deps gcc libffi-dev musl-dev \
 && pip install --no-cache-dir "Radicale[bcrypt] @ https://github.com/Kozea/Radicale/archive/refs/tags/${VERSION}.tar.gz" \
 && apk del .build-deps
COPY config /etc/radicale/config
COPY start.sh /start.sh
COPY --from=builder /tmp/users /etc/radicale/users
COPY --from=builder /usr/local/go/src/caldavserver/caldavserver /caldavserver
COPY flag.txt /flag.txt

# Add nginx
RUN apk add nginx
COPY default /etc/nginx/nginx.conf

# Add Calendar
RUN mkdir -p /var/lib/radicale/collections/collection-root/jrarj/default
COPY test.ics /var/lib/radicale/collections/collection-root/jrarj/default/test.ics
RUN echo '{"tag": "VCALENDAR"}' > /var/lib/radicale/collections/collection-root/jrarj/default/.Radicale.props

# Prepare permissions
RUN addgroup --gid 1001 -S radicale && \
    adduser -G radicale --shell /sbin/nologin --disabled-password -H --uid 1001 radicale && \
    chown -R radicale:radicale /var/lib/radicale/collections && \
    chmod +r /flag.txt && \
    chmod 700 /start.sh

# Run Radicale
CMD ["/start.sh"]
