FROM docker.io/library/golang:1.17-alpine AS builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w" -a -o zk-helper cmd/helper/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/zk-helper /usr/local/bin/zk-helper
USER 65532:65532
CMD ["/usr/local/bin/zk-helper"]