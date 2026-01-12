# Python to Go Migration Guide

## Overview

This repository contains a complete Go implementation of the S3 Resource Operator, migrated from the original Python version. Both implementations are included to allow for gradual migration and testing.

## File Structure

### Go Implementation (New)
```
cmd/
  main.go                    # Main entry point
pkg/
  backends/                  # Backend implementations
    backend.go              # Interface definition
    versitygw.go           # VersityGW implementation
    minio.go               # MinIO implementation
    garage.go              # Garage implementation
    backend_test.go        # Tests
  controller/               # Kubernetes controller
    controller.go          # Main controller logic
    controller_test.go     # Tests
  metrics/                  # Prometheus metrics
    metrics.go             # Metrics definitions
    metrics_test.go        # Tests
go.mod                     # Go module definition
Dockerfile.go              # Multi-stage Go build
Makefile                   # Build automation
```

### Python Implementation (Legacy)
```
src/                       # Python source code
requirements.txt           # Python dependencies
Dockerfile                 # Python image build
```

## Migration Benefits

### 1. **Performance**
- Go is compiled and significantly faster than Python
- Lower memory footprint
- Better concurrent processing with goroutines

### 2. **Deployment**
- Single static binary (no Python runtime needed)
- Smaller Docker images (~20MB vs ~200MB)
- Faster startup time

### 3. **Type Safety**
- Compile-time type checking
- Better IDE support and refactoring
- Fewer runtime errors

### 4. **Kubernetes Ecosystem**
- Go is the native language for Kubernetes
- Better client-go library support
- Standard patterns for operators

## Building the Go Version

### Prerequisites
- Go 1.23 or later
- Docker (for containerized builds)

### Local Build
```bash
# Download dependencies
go mod download

# Build binary
make build

# Run tests
make test

# Run locally (requires kubeconfig)
make run
```

### Docker Build
```bash
# Build Docker image
make docker-build

# Or manually
docker build -f Dockerfile.go -t s3-resource-operator:go .
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
make coverage
```

### Run Specific Package Tests
```bash
go test ./pkg/backends -v
go test ./pkg/controller -v
go test ./pkg/metrics -v
```

## Configuration

Both Python and Go versions use the same configuration:

### Environment Variables
- `S3_ENDPOINT_URL` - S3 endpoint URL
- `ROOT_ACCESS_KEY` - Root access key
- `ROOT_SECRET_KEY` - Root secret key
- `BACKEND_NAME` - Backend type (versitygw, minio, garage)
- `ANNOTATION_KEY` - Kubernetes annotation to watch (default: `s3-resource-operator.io/enabled`)

### Command-line Flags (Go only)
```bash
./operator \
  -kubeconfig=/path/to/kubeconfig \
  -metrics-port=8000 \
  -annotation-key=s3-resource-operator.io/enabled \
  -backend-name=versitygw \
  -s3-endpoint-url=http://s3.example.com \
  -root-access-key=admin \
  -root-secret-key=password
```

## Deployment

### Update Helm Chart

To use the Go version, update `helm/values.yaml`:

```yaml
image:
  repository: ghcr.io/runningman84/s3-resource-operator
  tag: "go-latest"  # or specific version
```

And update `helm/templates/deployment.yaml` to use `Dockerfile.go`:

```yaml
spec:
  containers:
  - name: operator
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
    # ... rest of config remains the same
```

### Side-by-Side Testing

You can run both versions simultaneously by deploying them in different namespaces:

```bash
# Deploy Python version
helm install s3-operator-python ./helm -n s3-operator-python --create-namespace

# Deploy Go version
helm install s3-operator-go ./helm -n s3-operator-go --create-namespace \
  --set image.tag=go-latest
```

## Key Differences

### 1. **Logging**
- **Python**: Uses standard `logging` module
- **Go**: Uses `klog/v2` (Kubernetes standard)

### 2. **Error Handling**
- **Python**: Exceptions
- **Go**: Explicit error returns

### 3. **Concurrency**
- **Python**: Threading
- **Go**: Goroutines and channels

### 4. **Dependencies**
- **Python**: boto3, kubernetes client, prometheus_client
- **Go**: aws-sdk-go, client-go, prometheus client_golang

## Migration Checklist

- [x] Backend interface and implementations
- [x] Kubernetes controller logic
- [x] Metrics and health endpoints
- [x] Signal handling and graceful shutdown
- [x] Configuration via env vars and flags
- [x] Docker image with multi-stage build
- [x] Basic unit tests
- [ ] Integration tests
- [ ] Update CI/CD workflows
- [ ] Update Helm chart defaults
- [ ] Performance benchmarking
- [ ] Documentation updates

## Performance Comparison

### Docker Image Size
- Python: ~200MB (python:3.14-slim base)
- Go: ~20MB (alpine base with static binary)

### Memory Usage (Estimated)
- Python: ~150-200MB
- Go: ~30-50MB

### Startup Time
- Python: ~2-3 seconds
- Go: ~0.1-0.5 seconds

## Troubleshooting

### Missing Dependencies
```bash
go mod download
go mod tidy
```

### Build Failures
```bash
# Clean and rebuild
make clean
make build
```

### Test Failures
```bash
# Run tests with verbose output
go test -v ./...
```

## Rollback Plan

If issues are encountered with the Go version:

1. Revert Helm chart to use Python image
2. Redeploy using Python Dockerfile
3. File issues in GitHub with details

## Next Steps

1. **Test thoroughly** in development environment
2. **Run side-by-side** in staging
3. **Monitor metrics** for both versions
4. **Gradual rollout** to production
5. **Deprecate Python version** once Go is stable

## Contributing

When contributing to the Go version:
- Follow Go best practices and idioms
- Add tests for new functionality
- Run `make fmt` and `make vet` before committing
- Update this migration guide as needed

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Kubernetes client-go](https://github.com/kubernetes/client-go)
- [AWS SDK for Go](https://docs.aws.amazon.com/sdk-for-go/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
