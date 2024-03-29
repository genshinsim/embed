FROM chromedp/headless-shell:latest 
WORKDIR /
COPY preview /preview
RUN ls -la
RUN chmod +x /preview
ENTRYPOINT ["/preview"]