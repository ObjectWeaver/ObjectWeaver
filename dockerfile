# Define build arguments
ARG BUILDKIT_SBOM_SCAN_STAGE=true
ARG TARGETOS
ARG TARGETARCH

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

WORKDIR /go/src/app
COPY . .

# Install garble, git, and download dependencies
RUN apk add --no-cache git && \
    go mod download

# Install ca-certificates using apk (Alpine's package manager)
RUN apk add --no-cache ca-certificates

# Build the application with garble for obfuscation
ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /go/bin/app

FROM scratch

# Copy the CA certificates from the build stage
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the obfuscated binary
COPY --from=build /go/bin/app /

# Copy static files into the container
COPY ./static /static

# Switch to a non-root user
USER 65534

CMD ["/app"]
