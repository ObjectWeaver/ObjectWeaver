ARG TARGETOS
ARG TARGETARCH

# Disable SBOM scanning on build stage (contains Alpine tooling we don't ship)
FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.23 AS build
ARG BUILDKIT_SBOM_SCAN_STAGE=false

WORKDIR /go/src/app
COPY . .

RUN apk add --no-cache git && \
    go mod download

RUN apk add --no-cache ca-certificates

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /go/bin/app

# Final minimal image - only this stage is scanned for vulnerabilities
FROM scratch
ARG BUILDKIT_SBOM_SCAN_STAGE=true

# --- OCI Standard Labels for AGPL-3.0 Compliance ---
LABEL org.opencontainers.image.licenses="AGPL-3.0-only"
LABEL org.opencontainers.image.source="https://github.com/objectweaver/objectweaver"
LABEL org.opencontainers.image.title="ObjectWeaver"
LABEL org.opencontainers.image.description="Object generation service licensed under AGPL-3.0"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /go/bin/app /

COPY ./static /static

# --- Embed License File (AGPL requirement) ---
COPY LICENSE.txt /usr/share/licenses/objectweaver/LICENSE

USER 65534

CMD ["/app"]
