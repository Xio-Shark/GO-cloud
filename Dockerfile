ARG SERVICE_NAME
FROM alpine:3.20.3
ARG SERVICE_NAME

RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S -u 10001 -G appgroup appuser

WORKDIR /app
COPY bin/linux-amd64/${SERVICE_NAME} /app/app

USER 10001
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/app/app", "healthcheck"]

ENTRYPOINT ["/app/app"]
