ARG BUILDKIT_SBOM_SCAN_STAGE=true
ARG TARGETOS
ARG TARGETARCH

FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.21 AS build

WORKDIR /go/src/app
COPY . .

RUN apk add --no-cache git && \
    go mod download

RUN apk add --no-cache ca-certificates

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -tags nolog -ldflags="-s -w" -o /go/bin/app

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /go/bin/app /

COPY ./static /static

USER 65534

CMD ["/app"]
