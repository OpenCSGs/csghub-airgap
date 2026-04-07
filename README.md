# CSGHub Airgap

A CLI tool for creating and deploying offline bundles for CSGHub in air-gapped environments. This tool runs as a command-line utility and does not require any runtime daemon or background service.

## Features

- **Create** offline bundles containing K3s, Helm charts, and container images
- **Deploy** CSGHub from offline bundles without internet access
- **Clean up** K3s and all services when needed
- **Multi-architecture** support (amd64, arm64)
- **Configurable** components (Longhorn, NVDP, etc.)
- **No daemon** - Pure CLI tool that runs on demand

## Prerequisites

- Go 1.21 or higher (for building)
- Make (optional, for using Makefile)
- Root/sudo access (for deployment and cleanup operations)

## Building

### Build for current system

```bash
make build
```

### Build for Linux (amd64 + arm64)

```shell
make build-linux
```

Or build individually:

```shell
make build-linux-amd64
make build-linux-arm64
```

### Build for macOS (Intel + Apple Silicon)

```shell
make build-darwin
```

Or build individually:

```shell
make build-darwin-amd64
make build-darwin-arm64
```

### Build for all platforms

```shell
make build-all
```

### Clean build artifacts

```shell
make clean
```

***Notes:** For default settings regarding charts, please refer to `internal/manifest/charts`*

## Installation

Make the binary executable:

```shell
sudo chmod +x ./csghub-airgap-linux-amd64
```

### Commands

#### Create Bundle

Create an offline bundle containing all necessary components:

```shell
sudo csghub-airgap create [options]
```

##### Global Options

| Flag       | Alias | Description                        | Default                            |
| :--------- | :---- | :--------------------------------- | :--------------------------------- |
| `--output` | `-o`  | Output directory for the bundle    | `./csghub-airgap-<date>`           |
| `--arch`   | `-a`  | Target architecture (amd64, arm64) | Depending on the local environment |
| `--debug`  | `-d`  | Enable debug logging               | `false`                            |

***Notes:** create cross-platform cannot download packages for runtime.* 

##### K3s Options

| Flag         | Description                                                  | Default |
| :----------- | :----------------------------------------------------------- | :------ |
| `--skip-k3s` | Skip K3s installation                                        | `false` |
| `--k3s-env`  | Extra environment variables for K3s (KEY=VALUE)              | `[]`    |
| `--k3s-arg`  | Extra CLI arguments for K3s (e.g., --k3s-arg=--default-runtime=nvidia) | `[]`    |

##### Charts Options

| Flag                 | Alias | Description                                                 | Default    |
| :------------------- | :---- | :---------------------------------------------------------- | :--------- |
| `--csghub-version`   | `-cv` | CSGHub chart version                                        | (required) |
| `--csghub-runtime`   |       | Pull CSGHub runtime images                                  | `false`    |
| `--longhorn-version` | `-lv` | Longhorn chart version                                      | `1.11.1`   |
| `--skip-longhorn`    |       | Skip Longhorn installation                                  | `false`    |
| `--nvdp-version`     | `-nv` | NVIDIA device plugin version                                | `0.19.0`   |
| `--skip-nvdp`        |       | Skip NVDP installation                                      | `false`    |
| `--set`              | `-s`  | Set Helm values (e.g., csghub.key=value, supports multiple) | `[]`       |

##### Examples

```shell
# Basic bundle creation
sudo csghub-airgap create

# Create bundle for ARM64 with specific versions
sudo csghub-airgap create \
  --arch arm64 \
  --output /opt/csghub-bundle \
  --csghub-version 1.17.0 \
  --longhorn-version 1.11.0

# Skip Longhorn and NVDP
sudo csghub-airgap create \
  --skip-longhorn \
  --skip-nvdp \
  --output /tmp/bundle \
  --csghub-version 2.0.0

# Set custom Helm values
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --set csghub.global.gateway.external.domain=csghub.example.io \
  --set longhorn.csi.replicaCount=3

# Include CSGHub runtime image
sudo csghub-airgap create \
  --csghub-runtime

# Add K3s custom arguments
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --k3s-arg "--disable=traefik"

# Add K3s environment variables
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --k3s-env "K3S_TOKEN=mysecrettoken" \
  --k3s-env "K3S_DATASTORE_ENDPOINT=mysql://..."
```

***Notes:** You can view the default parameter settings in csghub-aigap-\<date>/metadata.yaml. These parameters will be applied automatically during deployment unless you manually specify to override them.* 

#### Deploy Bundle

Deploy CSGHub from an offline bundle:

```shell
sudo csghub-airgap deploy [options]
```

##### Options

| Flag              | Alias | Description                                                 | Default                  |
| :---------------- | :---- | :---------------------------------------------------------- | :----------------------- |
| `--input`         | `-i`  | Input bundle directory                                      | `./csghub-airgap-<date>` |
| `--debug`         | `-d`  | Enable debug logging                                        | `false`                  |
| `--skip-k3s`      |       | Skip K3s installation                                       | `false`                  |
| `--k3s-env`       |       | Extra environment variables for K3s (KEY=VALUE)             | `[]`                     |
| `--k3s-arg`       |       | Extra CLI arguments for K3s                                 | `[]`                     |
| `--data`          | `-D`  | Specify data directory for K3s and Longhorn                 | (system default)         |
| `--csghub-domain` | `-cd` | Specify CSGHub domain                                       | `csghub.example.com`     |
| `--skip-longhorn` |       | Skip Longhorn installation                                  | `false`                  |
| `--skip-nvdp`     |       | Skip NVDP installation                                      | `true`                   |
| `--set`           | `-s`  | Set Helm values (e.g., csghub.key=value, supports multiple) |                          |

##### Examples

```shell
# Basic deployment
sudo csghub-airgap deploy --input /opt/csghub-bundle

# Deploy with custom domain and data directory
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --csghub-domain csghub.mycompany.com \
  --data /mnt/data  ## both for k3s and longhorn volumes

# Skip Longhorn deployment
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --skip-longhorn

# Enable NVDP (disabled by default)
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --skip-nvdp=false  ## This will add `--default-runtime=nvidia` to --k3s-args automatic

# Set custom Helm values during deployment
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --set csghub.global.persistence.size=200Gi

# Add K3s environment variables
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --k3s-env "K3S_NODE_NAME=server01" \
  --k3s-env "K3S_DATASTORE_CAFILE=/etc/ssl/certs/ca-certificates.crt"
```

#### Cleanup

Remove K3s and all deployed services:

```shell
sudo csghub-airgap cleanup [options]
```

##### Options

| Flag    | Alias | Description                                | Default |
| :------ | :---- | :----------------------------------------- | :------ |
| `--yes` | `-y`  | Skip confirmation and run cleanup directly | `false` |

##### Examples

```shell
# Interactive cleanup (with confirmation prompt)
sudo csghub-airgap cleanup

# Force cleanup without confirmation
sudo csghub-airgap cleanup --yes
```

## Complete Workflow Examples

### Example 1: Basic Air-Gap Deployment

#### On Internet-Connected Machine (Bundle Creation)

```shell
# Create bundle with all components
sudo csghub-airgap create

# Verify bundle contents
ls -la ./csghub-airgap-2026-04-01
```

#### Transfer to Air-Gapped Machine

```shell
# Copy the tarball to air-gapped machine via USB, SCP, etc.
scp ./csghub-airgap-2026-04-01 user@airgap-machine:/root/
```

#### On Air-Gapped Machine (Deployment)

```shell
# Deploy CSGHub
sudo csghub-airgap deploy \
  --csghub-domain csghub.internal.company.com
```

### Example 2: Production Deployment with Custom Storage

#### Bundle Creation

```shell
sudo csghub-airgap create \
  --output /opt/csghub-prod-bundle \
  --csghub-version 2.0.0 \
  --longhorn-version 1.11.1 \
  --set csghub.global.persistence.storageClass=longhorn \
  --set csghub.global.persistence.size=200Gi
```

#### Deployment

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-prod-bundle \
  --csghub-domain csghub.prod.company.com \
  --data /data
```

### Example 3: GPU-Accelerated Deployment

#### Bundle Creation with NVDP

```shell
sudo csghub-airgap create \
  --output /opt/csghub-gpu-bundle \
  --csghub-version 2.0.0 \
  --nvdp-version 0.19.0
```

#### Deployment with GPU Support

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-gpu-bundle \
  --csghub-domain csghub.gpu.company.com \
  --skip-nvdp=false
```

### Example 4: Minimal Deployment (No Longhorn, No NVDP)

#### Bundle Creation

```shell
sudo csghub-airgap create \
  --output /opt/csghub-minimal-bundle \
  --arch amd64 \
  --csghub-version 2.0.0 \
  --skip-longhorn \
  --skip-nvdp
```

#### Deployment

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-minimal-bundle \
  --skip-longhorn \
  --csghub-domain csghub.minimal.company.com
```

### Example 5: Cleanup and Re-deployment

```shell
# Clean up existing deployment
sudo csghub-airgap cleanup --yes

# Deploy fresh
sudo csghub-airgap deploy \
  --input /opt/csghub-prod-bundle \
  --csghub-domain csghub.company.com
```

## Troubleshooting

### Permission Denied

Most commands require root privileges. Always run with `sudo`:

```shell
sudo csghub-airgap [command]
```

### Architecture Mismatch

Ensure the bundle architecture matches your system:

```shell
# Check system architecture
uname -m

# Create bundle with correct architecture
sudo csghub-airgap create --arch $(uname -m) --csghub-version 2.0.0
```

### Debug Mode

Enable debug logging for detailed output:

```shell
sudo csghub-airgap create --debug --csghub-version 2.0.0
sudo csghub-airgap deploy --debug --input /path/to/bundle
```

### Cleanup Failures

If cleanup fails, you can manually remove K3s:

```shell
# Run the K3s uninstall script
/usr/local/bin/k3s-uninstall.sh

# Or remove K3s completely
/usr/local/bin/k3s-killall.sh
/usr/local/bin/k3s-uninstall.sh
```

### Bundle Creation Fails

Check disk space and permissions:

```shell
# Check available disk space
df -h

# Check write permissions in output directory
ls -ld /path/to/output/directory

# Ensure Docker/container runtime is running
systemctl status docker
```

### Deployment Fails

Verify bundle integrity and system requirements:

bash

```shell
# Check bundle metadata
cat /path/to/bundle/metadata.yaml

# Verify K3s is not already running
systemctl status k3s

# Check port availability
netstat -tulpn | grep -E '6443|8080'
```

### Common Error Messages

| Error                   | Solution                                                |
| :---------------------- | :------------------------------------------------------ |
| `permission denied`     | Run command with `sudo`                                 |
| `architecture mismatch` | Create bundle with correct `--arch` flag                |
| `bundle not found`      | Verify `--input` path is correct                        |
| `k3s already running`   | Run `csghub-airgap cleanup` first                       |
| `disk space full`       | Free up space or specify different `--output` directory |

## Security Notes

- This tool requires root privileges for deployment and cleanup operations
- Bundle files may contain sensitive configuration data - secure them appropriately
- Always verify bundle integrity before deployment in air-gapped environments
- The tool does not run any background daemons or services

## Uninstallation

To completely remove csghub-airgap:

bash

```shell
# Remove the binary
sudo rm /usr/local/bin/csghub-airgap

# Remove any created bundles (optional)
sudo rm -rf /opt/csghub-* ./csghub-airgap*

# Clean up any deployments
sudo csghub-airgap cleanup --yes 2>/dev/null || true
```
