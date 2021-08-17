# Start by building the application.
FROM golang:1.17-buster as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go mod download

RUN go build -o /go/bin/node-dns-exporter

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/node-dns-exporter /
ENTRYPOINT ["/node-dns-exporter"]
