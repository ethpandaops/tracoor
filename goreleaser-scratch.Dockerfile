FROM gcr.io/distroless/static-debian11:latest
COPY tracoor* /tracoor
ENTRYPOINT ["/tracoor"]
