# S3 Resource Operator

[![Test](https://github.com/runningman84/s3-resource-operator/actions/workflows/test.yml/badge.svg)](https://github.com/runningman84/s3-resource-operator/actions/workflows/test.yml)
[![Release](https://github.com/runningman84/s3-resource-operator/actions/workflows/release.yml/badge.svg)](https://github.com/runningman84/s3-resource-operator/actions/workflows/release.yml)
[![Build and Publish](https://github.com/runningman84/s3-resource-operator/actions/workflows/publish.yml/badge.svg)](https://github.com/runningman84/s3-resource-operator/actions/workflows/publish.yml)

A Kubernetes operator to manage S3 buckets and IAM users for S3-compatible object stores like VersityGW, MinIO, and Garage, declaratively using Kubernetes secrets.

## Overview

The S3 Resource Operator automates the lifecycle of S3 resources within a compatible object store (e.g., VersityGW, MinIO, Garage). By watching for specially annotated Kubernetes secrets, the operator can automatically:

- Create S3 buckets.
- Create IAM users.
- Assign ownership of a bucket to a newly created user.

This allows for a GitOps-friendly, declarative approach to managing basic S3 resources directly from your Kubernetes manifests.

## Features

- **Automated S3 Resource Provisioning**: Automatically creates S3 buckets and IAM users based on Kubernetes secrets.
- **Backend Support**: Supports multiple S3 backends, including `versitygw`, `minio`, and `garage`.
- **Bucket Ownership Management**: Ensures existing buckets are owned by the correct user, and changes the owner if necessary.
- **Dynamic Reconfiguration**: Watches for changes to secrets and updates resources accordingly.
- **Initial Sync**: On startup, the operator performs a full sync to ensure all declared resources are correctly configured.
- **Graceful Shutdown**: Properly handles SIGTERM and SIGINT signals for clean shutdown in Kubernetes environments.
- **Prometheus Metrics**: Exposes metrics on port 8000 for monitoring (secrets processed, errors, sync duration).
- **Configurable**: All settings, including S3 endpoint and credentials, are configurable via environment variables.
- **Helm Chart**: Comes with a Helm chart for easy deployment via OCI registry.
- **Multi-Architecture Support**: Docker images built for both AMD64 and ARM64 architectures (including Apple Silicon, AWS Graviton).
- **Automated Releases**: Semantic versioning and automated releases using Conventional Commits.
- **SBOM Generation**: Comprehensive Software Bill of Materials (SBOM) in SPDX and CycloneDX formats for supply chain security.
- **Vulnerability Scanning**: Automated security scanning with detailed reports for every release.

## Prerequisites

- A running Kubernetes cluster (v1.21+).
- [Helm v3+](https://helm.sh/docs/intro/install/) installed.
- A running instance of an S3-compatible object store (e.g., [VersityGW](https://github.com/versity/versitygw), [MinIO](https://min.io/)) accessible from the cluster.
- Admin credentials for your S3 instance.

## Installation

The operator is deployed using a Helm chart published to GitHub Container Registry (OCI).

### Option 1: Install with inline credentials

Install the chart directly from the OCI registry with credentials:

```sh
helm install s3-resource-operator oci://ghcr.io/runningman84/s3-resource-operator \
      --version 1.3.1 \
      --namespace s3-resource-operator \
      --create-namespace \
      --set operator.secret.data.S3_ENDPOINT_URL="http://<your-s3-service-endpoint>" \
      --set operator.secret.data.S3_ACCESS_KEY="<your-admin-access-key>" \
      --set operator.secret.data.S3_SECRET_KEY="<your-admin-secret-key>"
```

### Option 2: Use an existing secret

If you manage your S3 credentials externally (e.g., using External Secrets Operator, Sealed Secrets, or Vault), you can reference an existing secret:

```sh
# First, create your secret (example using kubectl)
kubectl create secret generic my-s3-credentials \
  --namespace s3-resource-operator \
  --from-literal=S3_ENDPOINT_URL="http://<your-s3-service-endpoint>" \
  --from-literal=S3_ACCESS_KEY="<your-admin-access-key>" \
  --from-literal=S3_SECRET_KEY="<your-admin-secret-key>"

# Install the chart referencing the existing secret
helm install s3-resource-operator oci://ghcr.io/runningman84/s3-resource-operator \
      --version 1.3.1 \
      --namespace s3-resource-operator \
      --create-namespace \
      --set operator.secret.create=false \
      --set operator.secret.name="my-s3-credentials"
```

### Option 3: Install from local chart

Or clone the repository and install locally:

```sh
git clone https://github.com/runningman84/s3-resource-operator.git
cd s3-resource-operator
helm install s3-resource-operator ./helm \
      --namespace s3-resource-operator \
      --create-namespace \
      --set operator.secret.data.S3_ENDPOINT_URL="http://<your-s3-service-endpoint>" \
      --set operator.secret.data.S3_ACCESS_KEY="<your-admin-access-key>" \
      --set operator.secret.data.S3_SECRET_KEY="<your-admin-secret-key>"
```

## Usage

To provision a new bucket and user, create a Kubernetes `Secret` with the required annotation and data fields.

The operator looks for secrets with the annotation `s3-resource-operator.io/enabled: "true"`.

### Example Secret

Here is an example of a secret that will trigger the operator to create a bucket named `my-app-backups` and a user with the access key `my-app-user`.

Create a file named `my-app-secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-app-s3-credentials
  namespace: my-app
  annotations:
    s3-resource-operator.io/enabled: "true"
type: Opaque
stringData:
  # Required fields
  bucket-name: "my-app-backups"
  access-key: "my-app-user"
  access-secret: "a-very-strong-and-long-password"

  # Optional fields
  endpoint-url: "http://<your-s3-service-endpoint>"
  role: "user"
  user-id: "1001"
  group-id: "1001"
```

Apply it to your cluster:

```sh
kubectl apply -f my-app-secret.yaml
```

Once applied, the operator will perform the following actions:
1.  Create an IAM user named `my-app-user`.
2.  Create an S3 bucket named `my-app-backups`.
3.  Change the owner of the `my-app-backups` bucket to `my-app-user`.

When you delete the secret, the operator will not delete the user or the bucket. This is a safety measure to prevent accidental data loss.

The secret should contain the following data fields:

- `bucket-name`: The name of the S3 bucket. **(Required)**
- `access-key`: The access key for the IAM user. **(Required)**
- `access-secret`: The secret key for the IAM user. **(Required)**
- `endpoint-url`: (Optional) The S3 endpoint URL. If provided, it must match the operator's `S3_ENDPOINT_URL` configuration.
- `role`: (Optional) The role to assign to the user.
- `user-id`: (Optional) The user ID to assign to the user.
- `group-id`: (Optional) The group ID to assign to the user.

### External Secrets

It is recommended to use a tool like the [External Secrets Operator](https://external-secrets.io/) to manage the secrets that this operator consumes. This allows you to store your S3 credentials in a secure secret store like Vault, AWS Secrets Manager, or Google Secrets Manager. An example of an `ExternalSecret` can be found in `crontrib/example-external-secret.yaml`.

## Development

To run the operator locally for development, you need Python 3.12+ and the required dependencies.

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/runningman84/s3-resource-operator.git
    cd s3-resource-operator
    ```

2.  **Install dependencies:**

    ```sh
    pip install -r requirements.txt
    ```

3.  **Set Environment Variables:**

    The operator reads its configuration from environment variables.

    ```sh
    export KUBECONFIG=~/.kube/config
    export S3_ENDPOINT_URL="http://<your-s3-endpoint>"
    export S3_ACCESS_KEY="<your-admin-access-key>"
    export S3_SECRET_KEY="<your-admin-secret-key>"
    export ANNOTATION_KEY="s3-resource-operator.io/enabled"
    export BACKEND_NAME="versitygw"
    ```

4.  **Run the operator:**

    ```sh
    python3 -m src.main
    ```

## Code Architecture

The operator follows a modular architecture with clear separation of concerns:

### Source Code Structure

```
src/
├── main.py           # Entry point and application initialization
├── operator.py       # Main operator logic and watch loop
├── secrets.py        # Kubernetes secret discovery and processing
├── metrics.py        # Prometheus metrics and health endpoints
├── utils.py          # Utility functions (Kubernetes API client, signing)
└── backends/         # S3 backend implementations
    ├── backend.py    # Abstract base class
    ├── versitygw.py  # VersityGW backend
    ├── minio.py      # MinIO backend
    └── garage.py     # Garage backend
```

### Module Responsibilities

#### `main.py` - Entry Point
- Application initialization and configuration
- Environment variable loading
- Signal handler setup for graceful shutdown
- Orchestrates startup of all components

#### `operator.py` - Operator Logic
- **Operator class**: Main orchestration and watch loop
- Secret lifecycle management (`handle_secret`)
- Periodic synchronization (`sync`)
- Kubernetes watch stream handling
- Prometheus metrics definitions:
  - `s3_operator_secrets_processed_total`
  - `s3_operator_errors_total`
  - `s3_operator_sync_duration_seconds`
  - `s3_operator_handle_secret_duration_seconds`

#### `secrets.py` - Secret Management
- **SecretManager class**: Kubernetes secret operations
- Secret discovery with annotation filtering
- Base64 decoding of secret data
- Secret validation and processing

#### `metrics.py` - Observability
- **MetricsServer class**: HTTP server for monitoring
- Prometheus metrics endpoint (`/metrics`)
- Health check endpoint (`/healthz`)
- Threaded HTTP server for non-blocking operation

#### `utils.py` - Utilities
- Kubernetes API client initialization (`get_k8s_api`)
- AWS Signature V4 request signing
- Signed request execution
- Reusable helper functions

#### `backends/` - Backend Implementations
- Abstract `Backend` base class defining the interface
- Backend-specific implementations for:
  - **VersityGW**: Full support (create bucket/user, ownership)
  - **MinIO**: Planned support
  - **Garage**: Planned support
- Pluggable architecture for easy backend addition

### Test Structure

Tests are organized to mirror the source code structure:

```
tests/
├── conftest.py         # Pytest fixtures and configuration
├── test_operator.py    # Tests for Operator class
├── test_secrets.py     # Tests for SecretManager class
├── test_metrics.py     # Tests for Prometheus metrics
├── test_health.py      # Tests for health endpoints
├── test_shutdown.py    # Tests for graceful shutdown
├── test_backends.py    # Tests for backend implementations
└── test_config.py      # Tests for configuration loading
```

Each test module corresponds to its source module, making it easy to locate and maintain tests.

### Design Principles

- **Single Responsibility**: Each module has a focused purpose
- **Separation of Concerns**: Clear boundaries between components
- **Testability**: Modular design enables comprehensive unit testing
- **Extensibility**: New backends can be added without modifying core logic
- **Observability**: Built-in metrics and health checks

## CI/CD

This project uses GitHub Actions for continuous integration and deployment with a fully automated release pipeline:

### Workflows

1. **Test** (`.github/workflows/test.yml`)
   - Runs on: Push to `develop` branch and all pull requests
   - Executes: Python tests, Helm chart validation, Docker build verification, Trivy security scan
   - Can be: Called by other workflows (`workflow_call`) or run manually

2. **Commit Validation** (`.github/workflows/commitlint.yml`)
   - Runs on: All pull requests
   - Validates: All commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) format
   - Posts: Helpful feedback on PRs if validation fails

3. **Release** (`.github/workflows/release.yml`)
   - Runs on: Push to `main` branch
   - Flow (3 sequential jobs):
     - **Test job**: Calls test workflow (reuses `test.yml`)
     - **Release job**: Runs semantic-release to create version and tag
       - Analyzes commits to determine version bump (major/minor/patch)
       - Updates `CHANGELOG.md`, `helm/Chart.yaml`, and `helm/values.yaml`
       - Creates git tag and GitHub release
     - **Publish job**: Calls Build and Publish workflow (only if release was created)
   - Requirements: All commits must follow [Conventional Commits](https://www.conventionalcommits.org/) format

4. **Build and Publish** (`.github/workflows/publish.yml`)
   - Triggered by: Release workflow (automatically via `workflow_call`)
   - **Cannot be triggered manually** - Security measure to prevent accidental releases
   - Builds:
     - **Multi-architecture Docker images** using native runners:
       - `linux/amd64` - Built natively on x86_64 runners (`ubuntu-24.04`)
       - `linux/arm64` - Built natively on ARM64 runners (`ubuntu-24.04-arm`)
     - Publishes to: `ghcr.io/runningman84/s3-resource-operator`
     - Tags: `latest`, `1.0.1`, `1.0`, `1`
   - **Security & Compliance**:
     - Generates comprehensive SBOMs using Syft (SPDX & CycloneDX formats)
     - Platform-specific SBOMs for AMD64 and ARM64
     - Scans for vulnerabilities using Grype
     - Uploads all SBOMs and scan reports to GitHub release
   - **Helm chart** package and publish to OCI registry
     - Publishes to: `oci://ghcr.io/runningman84/s3-resource-operator`

5. **Sync Main to Develop** (`.github/workflows/sync.yml`)
   - Runs on: Push to `main` branch (after releases)
   - Automatically syncs changes from `main` back to `develop`
   - Ensures `develop` stays up-to-date with:
     - CHANGELOG.md updates
     - Version bumps in Chart.yaml and values.yaml
     - Any hotfixes or patches applied to main
   - Uses `[skip ci]` to prevent triggering unnecessary workflows

### Multi-Architecture Images

Docker images are built natively for both AMD64 and ARM64 architectures without using slow QEMU emulation. This means:
- ✅ **Fast builds** - Native compilation on each architecture (~3x faster than QEMU)
- ✅ **Apple Silicon** - Run natively on M1/M2/M3 Macs
- ✅ **AWS Graviton** - Optimized for ARM-based cloud instances
- ✅ **Intel/AMD** - Traditional x86_64 servers

Docker automatically pulls the correct architecture for your platform:
```bash
# Works on any architecture
docker pull ghcr.io/runningman84/s3-resource-operator:latest
```

### SBOM & Supply Chain Security

Each release includes comprehensive Software Bill of Materials (SBOM) and vulnerability reports:

#### SBOM Generation

Complete Software Bill of Materials (SBOM) generated using [Syft](https://github.com/anchore/syft):

- **What it contains**: Complete runtime environment
  - Operating system packages (Debian, Alpine, etc.)
  - Python runtime and all installed packages
  - System libraries and dependencies
  - Everything actually deployed in production
- **Formats**: SPDX and CycloneDX (JSON)
- **Platform-specific**: Separate SBOMs for AMD64 and ARM64
- **Combined SBOM**: Multi-platform overview
- **Use cases**:
  - Runtime security scanning
  - Deployment compliance
  - License compliance
  - Supply chain security
  - CycloneDX format compatible with dependency tracking tools

#### Vulnerability Scanning
- **Automated Scanning**: Every release is scanned for known vulnerabilities using [Grype](https://github.com/anchore/grype)
- **Multi-Platform**: Separate scans for AMD64 and ARM64 architectures
- **Report Formats**:
  - Human-readable table format for quick review
  - JSON format for automation and integration
  - SARIF format for GitHub Code Scanning integration
- **Available as Release Assets**: All reports attached to GitHub releases

#### Accessing Security Information

All SBOM and vulnerability reports are attached to each GitHub release:

```bash
# Download SBOMs for a specific release
gh release download v1.2.0 --pattern 'sbom-*.json'

# Download vulnerability reports
gh release download v1.2.0 --pattern 'vulnerability-report*'
```

**Files included with each release:**
- `sbom-spdx.json` - Combined SBOM (SPDX format)
- `sbom-cyclonedx.json` - Combined SBOM (CycloneDX format)
- `sbom-amd64-spdx.json` - AMD64-specific SBOM
- `sbom-amd64-cyclonedx.json` - AMD64-specific SBOM
- `sbom-arm64-spdx.json` - ARM64-specific SBOM
- `sbom-arm64-cyclonedx.json` - ARM64-specific SBOM
- `vulnerability-report-amd64.txt` - AMD64 vulnerability scan (table)
- `vulnerability-report-arm64.txt` - ARM64 vulnerability scan (table)
- `vulnerability-report.json` - Vulnerability scan (JSON)
- `vulnerability-report.sarif` - Vulnerability scan (SARIF)

#### Integration with Security Tools

The generated SBOMs and reports can be integrated with:
- **Dependency Track**: Import CycloneDX SBOMs for continuous monitoring
- **GitHub Dependency Graph**: SPDX format supported
- **OWASP Dependency-Check**: Compatible with both formats
- **Snyk, Anchore, Aqua**: Standard SBOM formats supported
- **GitHub Code Scanning**: SARIF reports for vulnerability visibility
### Release Process

The release process is fully automated using semantic versioning:

1. **Development**: Create PRs against `develop` branch
   - All commits validated with commitlint
   - Tests run automatically

2. **Merge to Main**: When ready to release
   ```bash
   # Ensure your commits follow Conventional Commits format:
   # feat: adds new feature (minor version bump)
   # fix: bug fix (patch version bump)
   # feat!: breaking change (major version bump)
   git checkout main
   git merge develop
   git push
   ```

3. **Automatic Release & Publish**:
   - Tests run first (if tests fail, no release)
   - Semantic-release analyzes commit messages
   - Version is determined automatically
   - Changelog is generated
   - Git tag and GitHub release created
   - **Build and Publish workflow is automatically triggered**
   - Multi-architecture Docker images built (AMD64 + ARM64)
   - Helm chart packaged and published

4. **Result**: New version available within minutes at:
   - Docker: `ghcr.io/runningman84/s3-resource-operator:1.2.0`
   - Helm: `oci://ghcr.io/runningman84/s3-resource-operator --version 1.2.0`


### Complete Automation Flow

```
Developer Pushes to main
         ↓
   Release Workflow (3 Jobs)
    ┌──────────────────────┐
    │ Job 1: Test          │
    │ - Call test.yml      │
    │ - Python tests       │
    │ - Helm validation    │
    │ - Docker build       │
    │ - Trivy scan         │
    └──────────┬───────────┘
               ↓
    ┌──────────────────────┐
    │ Job 2: Release       │
    │ - Semantic Release   │
    │ - Determine Version  │
    │ - Update Files       │
    │ - Create Tag         │
    │ - Create Release     │
    └──────────┬───────────┘
               ↓
    ┌──────────────────────┐
    │ Job 3: Publish       │
    │ - Call publish.yml   │
    │   (if release created)│
    └──────────┬───────────┘
               ↓
   Build and Publish Workflow
    ├─ Build AMD64 Image (native ubuntu-24.04)
    ├─ Build ARM64 Image (native ubuntu-24.04-arm)
    ├─ Create Multi-Arch Manifest
    ├─ Tag Images (version, major, latest)
    ├─ Generate SBOMs (SPDX & CycloneDX)
    ├─ Scan for Vulnerabilities (Grype)
    ├─ Upload SBOMs & Reports to Release
    └─ Package & Publish Helm Chart
         ↓
   Sync Main to Develop Workflow
    └─ Merge main → develop (includes CHANGELOG, versions)
         ↓
   🎉 Release Complete!
```

### Commit Message Validation

All PRs are automatically validated to ensure commit messages follow the Conventional Commits format. The commitlint workflow will comment on PRs with guidance if validation fails.

## Configuration

The operator can be configured using the following environment variables:

| Environment Variable      | Description                                                                 | Default                        |
| ------------------------- | --------------------------------------------------------------------------- | ------------------------------ |
| `ANNOTATION_KEY`          | The annotation key to look for on secrets.                                  | `s3-resource-operator.io/enabled` |
| `S3_ENDPOINT_URL`         | The URL of the S3 endpoint.                                                 | (required)                     |
| `S3_ACCESS_KEY`           | The access key for the S3 endpoint (for the operator itself).               | (required)                     |
| `S3_SECRET_KEY`           | The secret key for the S3 endpoint (for the operator itself).               | (required)                     |
| `BACKEND_NAME`            | The name of the S3 backend to use (`versitygw`, `minio`, `garage`).         | `versitygw`                    |

## Monitoring

The operator exposes Prometheus metrics and a health check endpoint on port 8000:

### Health Check
- **Endpoint**: `/healthz`
- **Purpose**: Kubernetes liveness and readiness probes
- **Response**: `{"status":"healthy"}`

```sh
kubectl port-forward -n s3-resource-operator deployment/s3-resource-operator 8000:8000
curl http://localhost:8000/healthz
```

### Metrics
- **Endpoint**: `/metrics`
- **Available Metrics**:
  - `s3_operator_secrets_processed_total`: Total number of secrets processed
  - `s3_operator_errors_total`: Total number of errors encountered
  - `s3_operator_sync_duration_seconds`: Duration of sync cycles (histogram)
  - `s3_operator_handle_secret_duration_seconds`: Duration of handling individual secrets (histogram)

To access metrics:
```sh
kubectl port-forward -n s3-resource-operator deployment/s3-resource-operator 8000:8000
curl http://localhost:8000/metrics
```

### Prometheus Operator
If you're using the Prometheus Operator, you can enable automatic service discovery by setting `serviceMonitor.enabled=true` in your Helm values:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our development process, coding standards, and how to submit pull requests.

### 📝 Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automated releases and changelog generation. **All commits must follow this format:**

```
<type>: <description>

[optional body]

[optional footer]
```

**Valid types:**
- `feat`: A new feature (triggers minor version bump)
- `fix`: A bug fix (triggers patch version bump)
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring without functionality changes
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Changes to build process or auxiliary tools
- `ci`: CI configuration changes
- `build`: Changes to build system or dependencies

**Examples:**
```
feat: add support for bucket versioning
fix: correct owner change logic for garage backend
docs: update installation instructions
test: add tests for graceful shutdown
chore(deps): update kubernetes dependencies
```

**Breaking Changes:**
For breaking changes, add `!` after the type or include `BREAKING CHANGE:` in the footer:
```
feat!: remove deprecated API endpoint

BREAKING CHANGE: The /api/v1/legacy endpoint has been removed.
```

💡 **Tip:** When creating pull requests, you can squash commits and write a proper conventional commit message during merge.

- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and contribution guidelines
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community standards and expectations
- **[Security Policy](SECURITY.md)** - How to report security vulnerabilities

This project uses [Renovate](https://github.com/renovatebot/renovate) to keep dependencies up-to-date. Renovate will automatically create pull requests for outdated dependencies.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
