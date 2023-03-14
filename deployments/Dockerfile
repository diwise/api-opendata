FROM --platform=$BUILDPLATFORM golang:1.20 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

#RUN go test -v ./...

WORKDIR /app/cmd/api-opendata

ARG TARGETARCH
ARG TARGETOS
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /opt/diwise

LABEL "org.opencontainers.image.source" = "https://github.com/diwise/api-opendata"

COPY --from=builder --chown=1001 /app/cmd/api-opendata/api-opendata /opt/diwise
COPY --chown=1001 api/openapi-spec/openapi.json /opt/diwise/openapi.json
COPY --chown=1001 assets/dcat.rdf /opt/diwise/datasets/dcat.rdf

RUN chown -R 1001 /opt/diwise

EXPOSE 8080
USER 1001

ENTRYPOINT ["/opt/diwise/api-opendata", "-rdffile", "/opt/diwise/datasets/dcat.rdf", "-oas", "/opt/diwise/openapi.json"]
