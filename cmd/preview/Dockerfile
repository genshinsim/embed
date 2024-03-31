FROM alpine:3.16.3
WORKDIR /
COPY preview /preview
RUN ls -la
RUN chmod +x /preview
ENTRYPOINT ["/preview"]