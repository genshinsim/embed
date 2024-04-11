FROM alpine:3.16.3
RUN apk add --no-cache git
WORKDIR /
COPY init.sh /init.sh
RUN chmod +x /init.sh
COPY preview /preview
RUN chmod +x /preview
RUN ls -la
ENTRYPOINT ["/init.sh"]