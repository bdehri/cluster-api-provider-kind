# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY src/ src/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go


RUN curl -Lo /tmp/kind https://kind.sigs.k8s.io/dl/v0.14.0/kind-linux-amd64
RUN chmod +x /tmp/kind
RUN curl https://download.docker.com/linux/static/stable/x86_64/docker-19.03.1.tgz -o /tmp/docker.tar.gz
RUN tar xzvf /tmp/docker.tar.gz -C /tmp

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:latest
WORKDIR /
COPY --from=builder /workspace/manager .

COPY --from=builder /tmp/kind /usr/local/bin/kind
COPY --from=builder /tmp/docker /usr/local/bin/

CMD dockerd &  ; /manager