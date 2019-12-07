FROM golang:1.13 AS builder
WORKDIR /go/src/gandi-dyndns
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -o /gandi-dyndns .

FROM scratch
COPY --from=builder /gandi-dyndns /gandi-dyndns
ENTRYPOINT ["/gandi-dyndns"]
