FROM gcr.io/distroless/static:nonroot
COPY webhook /
ENTRYPOINT ["/webhook"]