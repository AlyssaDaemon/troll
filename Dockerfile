FROM golang:alpine as builder

ENV PROJECT_NAME=github.com/alyssadaemon/troll
ENV WORKDIR=${GOPATH}/src/${PROJECT_NAME}
ENV GO111MODULE=on
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR ${WORKDIR}
COPY . .

RUN adduser -D -g '' appuser
RUN apk add --no-cache git ca-certificates && update-ca-certificates
RUN go mod download && go mod verify
RUN go build -ldflags="-w -s" -o /go/bin/troll cmd/troll/main.go

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/troll /go/bin/troll

USER appuser

ENTRYPOINT ["/go/bin/troll"]