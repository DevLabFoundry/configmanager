ARG Version
ARG Revision

FROM docker.io/golang:1-trixie AS builder

ARG Version=0.0.1
ARG Revision=beta01

WORKDIR /app

COPY ./ /app
RUN CGO_ENABLED=0 go build -mod=readonly -buildvcs=false \
    -ldflags="-s -w -X \"github.com/DevLabFoundry/configmanager/v3/cmd/configmanager.Version=${Version}\" -X \"github.com/DevLabFoundry/configmanager/v3/cmd/configmanager.Revision=${Revision}\" -extldflags -static" \
    -o bin/configmanager cmd/main.go

FROM docker.io/alpine:3

COPY --from=builder /app/bin/configmanager /usr/bin/configmanager

RUN chmod +x /usr/bin/configmanager

RUN adduser -D -s /bin/sh -h /home/configmanager configmanager

USER configmanager

ENTRYPOINT ["configmanager"]
