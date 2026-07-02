FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine AS builder

WORKDIR /build
COPY api ./api

WORKDIR /build/grype-server
COPY grype-server/go.* ./
RUN go mod download

# Copy and build backend code
COPY grype-server .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /app/grype-server ./cmd/grype-server/main.go \
    && mkdir -p /data

FROM dhi.io/alpine-base:3.23

COPY --from=builder --chown=1000:1000 /app /app
COPY --from=builder --chown=1000:1000 /data /data

ENV DB_ROOT_DIR=/data

USER 1000:1000

ENTRYPOINT ["/app/grype-server"]

CMD ["run"]

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
LABEL org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.name="grype-server" \
    org.label-schema.description="Running Grype scanner as a K8s server" \
    org.label-schema.url="https://github.com/brennoo/grype-server" \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vcs-url="https://github.com/brennoo/grype-server"

### Required OpenShift Labels
ARG IMAGE_VERSION
LABEL name="grype-server" \
      vendor="brennoo" \
      version=${IMAGE_VERSION} \
      release=${IMAGE_VERSION} \
      summary="Grype scanner as a K8s server" \
      description="Running Grype scanner as a K8s server"
