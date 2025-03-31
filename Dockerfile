FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o bin/helm-values-manager cmd/helm-values-manager/main.go

FROM alpine:3.18

RUN apk add --no-cache ca-certificates curl bash git openssl && \
  adduser -D -u 1000 appuser && \
  mkdir -p /app/values-analysis && \
  chown -R appuser:appuser /app

# Install specific Helm version
ARG HELM_VERSION=v3.12.3
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
  chmod 700 get_helm.sh && \
  DESIRED_VERSION=${HELM_VERSION} ./get_helm.sh && \
  rm get_helm.sh

WORKDIR /app
COPY --from=builder /app/bin/helm-values-manager /app/bin/helm-values-manager
COPY scripts/generate-examples.sh /app/scripts/
COPY plugin.yaml README.md /app/

RUN chmod +x /app/bin/helm-values-manager /app/scripts/generate-examples.sh && \
  chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["/app/bin/helm-values-manager"]
CMD ["-h"]
