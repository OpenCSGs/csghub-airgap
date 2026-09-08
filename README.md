# CSGHub Airgap

A CLI tool for creating and deploying offline bundles for CSGHub in air-gapped environments. Run `create` on a machine with internet access to build the bundle, then `deploy` on the offline machine. This tool runs as a command-line utility and does not require any runtime daemon or background service.

## Features

- **Create** offline bundles containing K3s, Helm charts, and container images
- **Deploy** CSGHub from offline bundles without internet access
- **Clean up** K3s and all services when needed
- **Multi-architecture** support (amd64, arm64)
- **Selective components** with a unified `--skip` flag (charts, images, runtime images, packages, tar, K3s)
- **Interactive runtime image selection** (TUI) during `create`
- **List runtime images** with category and compute type filters
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

## Commands

| Command        | Description                                            |
| :------------- | :----------------------------------------------------- |
| `create`       | Create an offline bundle on an internet-connected machine |
| `deploy`       | Deploy CSGHub from an offline bundle on an offline machine |
| `cleanup`      | Remove K3s and all deployed services                   |
| `list-images`  | List available runtime images before creating a bundle |

### Create Bundle

Create an offline bundle containing all necessary components:

```shell
sudo csghub-airgap create [options]
```

#### Options

| Flag              | Alias | Description                                                          | Default                  |
| :---------------- | :---- | :------------------------------------------------------------------- | :----------------------- |
| `--output`        | `-o`  | Output directory for the bundle                                      | `./csghub-airgap-<date>` |
| `--arch`          |       | Target architecture (amd64, arm64)                                   | Current system           |
| `--debug`         | `-d`  | Enable debug logging                                                 | `false`                  |
| `--proxy`         |       | Use ghfast.top proxy prefix for GitHub downloads                     | `false`                  |
| `--force-pull`    |       | Skip the local image cache (containerd cache + file cache) and always re-pull from the registry | `false` |
| `--no-pkg-cache`  |       | Skip the cached NVIDIA package directory and always re-download apt/dnf/zypper packages | `false` |
| `--k3s-env`       |       | Extra environment variables for K3s (KEY=VALUE)                      | `[]`                     |
| `--k3s-arg`       |       | Extra CLI arguments for K3s (e.g., `--k3s-arg=--default-runtime=nvidia`) | `[]`                |
| `--skip`          |       | Skip components: comma-separated chart names (csghub, longhorn, nvdp) and/or kinds (k3s, charts, image, runtime, package, tar) |   |
| `--quiet`         | `-q`  | Skip interactive prompts; runtime image selection is skipped (no images selected) | `false` |
| `--dry-run`       |       | Print the operations that would be performed (downloads, image exports, packing) without executing them | `false` |
| `--set`           | `-s`  | Set Helm values (e.g., csghub.key=value, supports multiple)          | `[]`                     |

#### Execution Flow

1. **Architecture check** - verify the current platform can run (creating on macOS is supported, see notes below).
2. **NVIDIA package download** - only when the target arch matches the host, the nvdp chart is enabled, and `--skip package` is not set.
3. **K3s download** - k3s binary, airgap images, install script, helm binary (unless `--skip k3s`).
4. **Per-chart processing** (each enabled chart):
   - Pull the Helm chart (unless `--skip charts`; if `--skip charts` but the chart file is missing locally, it is still pulled).
   - Export the chart's main images (unless `--skip image`).
   - For the `csghub` chart: show the runtime image selection UI, then export the selected runtime images (unless `--skip runtime` or `--quiet`).
5. **Generate `metadata.yaml`** recording the bundle configuration.
6. **Pack** - pack the whole output directory into `<dirname>.tar` (unless `--skip tar`).

> **Runtime image selection timing**: runtime images are selected and exported after the csghub main image export, not at program start.

#### Runtime Image Selection UI

After the csghub main image export, the program reads csghub-server's runtime image config (default from GitHub `main` branch) and shows a TUI for you to select:

```
Select runtime images (space=toggle, a=all, A=none, enter=confirm, empty=skip)
```

- `space` toggle the current item
- `a` select all
- `A` select none
- `enter` confirm (selecting nothing = skip, i.e. no runtime images bundled)
- `q` / `Esc` cancel (also skips runtime images)

#### Examples

```shell
# Basic bundle creation (interactive runtime image selection)
sudo csghub-airgap create

# Specify output directory and use proxy
sudo csghub-airgap create \
  --output /opt/csghub-bundle \
  --proxy

# Create an arm64 bundle
sudo csghub-airgap create --arch arm64

# No runtime image selection (non-interactive)
sudo csghub-airgap create --quiet

# Skip runtime images entirely
sudo csghub-airgap create --skip runtime

# Export only runtime images (skip main chart images too)
sudo csghub-airgap create --skip k3s,charts,package,tar,image

# Image-only mode: skip k3s / charts / packages (shows the main image selection UI)
sudo csghub-airgap create --skip k3s,charts,package

# Keep the plain directory (no .tar), useful with deploy --skip tar
sudo csghub-airgap create --skip tar

# Skip specific charts
sudo csghub-airgap create --skip longhorn
sudo csghub-airgap create --skip longhorn,nvdp

# Force re-pull all images (ignore local cache)
sudo csghub-airgap create --force-pull

# Custom Helm values and K3s args
sudo csghub-airgap create \
  --set csghub.global.gateway.external.domain=csghub.mycompany.com \
  --set longhorn.persistence.defaultClassReplicaCount=3 \
  --k3s-arg "--disable=servicelb"

# Extra K3s environment variables
sudo csghub-airgap create \
  --k3s-env "K3S_NODE_NAME=server01" \
  --k3s-env "K3S_TOKEN=mysecrettoken"
```

#### Bundle Directory Structure

After `create` finishes, besides the directory below, a `csghub-airgap-<date>.tar` archive of the whole directory is generated next to it for easy transfer. During `deploy` you can point `--input` at the archive (auto-unpacked) or at an already-unpacked directory (with `--skip tar`).

```
csghub-airgap-<date>/
├── metadata.yaml                  # bundle config (auto-applied at deploy)
├── k3s/
│   ├── bin/k3s
│   ├── images/k3s-airgap-images-<arch>.tar.zst
│   ├── scripts/k3s-install.sh
│   ├── scripts/k3s-patch.sh
│   └── helm/helm-v3.20.1-linux-<arch>.tar.gz
├── packages/                      # NVIDIA container toolkit packages (Linux only)
├── csghub/
│   ├── charts/csghub-<version>.tgz
│   └── images/csghub-<version>.tar
├── csghub-runtime/                # only when runtime images were selected
│   └── images/csghub-runtime-<version>.tar
├── longhorn/
│   ├── charts/longhorn-<version>.tgz
│   └── images/longhorn-<version>.tar
└── nvidia-device-plugin/
    ├── charts/nvidia-device-plugin-<version>.tgz
    └── images/nvidia-device-plugin-<version>.tar
```

#### Notes

- **Cross-architecture**: NVIDIA packages are not downloaded when the target arch differs from the host; fetch them manually on a Linux machine (e.g. `apt/dnf/zypper install nvidia-container-toolkit=...`).
- **Creating on macOS**: supported, but NVIDIA packages cannot be downloaded on non-Linux hosts; handle them manually.
- **`--flannel-iface`**: detected from the build machine (e.g. `en0` on macOS); if not overridden with `--k3s-arg` at deploy time, it is replaced with the target machine's default NIC.

### Deploy Bundle

Deploy CSGHub from an offline bundle (requires root):

```shell
sudo csghub-airgap deploy [options]
```

#### Options

| Flag              | Alias | Description                                                | Default                  |
| :---------------- | :---- | :--------------------------------------------------------- | :----------------------- |
| `--input`         | `-i`  | Input bundle directory (or the `.tar` archive)             | `./csghub-airgap-<date>` |
| `--debug`         | `-d`  | Enable debug logging                                       | `false`                  |
| `--proxy`         |       | Use ghfast.top proxy prefix for GitHub downloads           | `false`                  |
| `--k3s-env`       |       | Extra environment variables for K3s (KEY=VALUE)            | `[]`                     |
| `--k3s-arg`       |       | Extra CLI arguments for K3s                                | `[]`                     |
| `--data`          | `-D`  | Specify data directory for K3s and Longhorn                | (system default)         |
| `--domain`        |       | Specify CSGHub domain                                      | `csghub.example.com`     |
| `--quiet`         | `-q`  | Skip interactive prompts; multipathd is auto-disabled (no prompt) | `false`          |
| `--dry-run`       |       | Print the operations that would be performed (incl. k3s install command, chart install) without executing them | `false` |
| `--skip`          |       | Same as `create`: `k3s`, `charts`, `image`, `runtime`, `package`, `tar` and chart names |   |
| `--set`           | `-s`  | Set Helm values (e.g., csghub.key=value, supports multiple) | `[]`                   |

#### Execution Flow and Preflight

1. **System check** - must be Linux; the bundle architecture must match the target machine.
2. **NVIDIA GPU detection** - auto-disable the nvdp chart (and package install) when the host has no NVIDIA GPU.
3. **NVIDIA package install** - when nvdp is enabled and `--skip package` is not set.
4. **Kernel parameters** - write required sysctl parameters (skipped with `--skip k3s`, i.e. an own cluster).
5. **K3s install** - install k3s (unless `--skip k3s`).
6. **multipathd preflight** - when multipathd is running, prompt to stop and disable it (auto under `--quiet`). Longhorn conflicts with multipath and may hang PVC mounting.
7. **Per-chart deploy** (each enabled chart):
   - Import main images into containerd (unless `--skip image`).
   - For the `csghub` chart: import runtime images (unless `--skip runtime`; only when runtime images were selected at create time).
   - Install or upgrade the chart (unless `--skip charts`).
8. **K3s patch** - apply the domain and other config (writes `/etc/hosts` and patches the cluster CoreDNS). When k3s was not installed by this tool (`--skip k3s`, own cluster), prompts for confirmation first; `--quiet` applies it by default.

#### Examples

```shell
# Basic deployment
sudo csghub-airgap deploy --input /opt/csghub-bundle

# Specify domain and data directory
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --domain csghub.mycompany.com \
  --data /mnt/data

# Non-interactive deploy (auto-disables multipathd)
sudo csghub-airgap deploy --input /opt/csghub-bundle --quiet

# Skip K3s install (only import images and install charts)
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip k3s

# Skip all image imports (images already loaded another way)
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip image

# Skip runtime image import
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip runtime

# Skip specific charts
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip longhorn
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip nvdp

# Override Helm values at deploy time
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --set csghub.global.persistence.size=200Gi

# Extra K3s args and env
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --k3s-arg "--default-runtime=nvidia" \
  --k3s-env "K3S_NODE_NAME=server01"
```

### Cleanup

Remove K3s and all deployed services:

```shell
sudo csghub-airgap cleanup [options]
```

| Flag    | Alias | Description                                | Default |
| :------ | :---- | :----------------------------------------- | :------ |
| `--yes` | `-y`  | Skip confirmation and run cleanup directly | `false` |

```shell
# Interactive cleanup (confirmation prompt + 10s countdown, Ctrl+C to abort)
sudo csghub-airgap cleanup

# Force cleanup without confirmation
sudo csghub-airgap cleanup --yes
```

> ⚠️ **Warning**: cleanup removes the **entire k3s cluster on this machine and ALL its data** (not just CSGHub resources), including `/var/lib/rancher/k3s` (PVCs, Longhorn volumes), `/var/lib/kubelet`, the k3s binary and systemd units. **A k3s you installed yourself on this host will be deleted too** — make sure this is intended.
>
> Interactive mode requires typing `yes` to confirm, then waits through a 10-second countdown (`Ctrl+C` to cancel). This effectively runs `/usr/local/bin/k3s-uninstall.sh`. If it fails, run that script manually.

### List Runtime Images

List available runtime images from csghub-server configs before creating a bundle (no root needed):

```shell
csghub-airgap list-images [options]
```

| Flag           | Alias | Description                                                  | Default   |
| :------------- | :---- | :----------------------------------------------------------- | :-------- |
| `--source`     |       | Where to fetch configs: `github`, `local`                    | `github`  |
| `--version`    | `-v`  | Git branch or tag for csghub-server (e.g., `main`, `v1.2.0`) | `main`    |
| `--local-path` |       | Local directory containing csghub-server configs (used with `--source local`) | `.`       |
| `--category`   | `-c`  | Filter by category (can specify multiple): `evaluation`, `finetune`, `inference`, `job`, `notebook`, `space` | (all) |
| `--compute`    |       | Filter by compute type (comma-separated): `cpu`, `gpu`, `dcu` | (all)     |
| `--proxy`      |       | Use ghfast.top proxy prefix for GitHub downloads             | `false`   |
| `--format`     | `-f`  | Output format: `table`, `json`                               | `table`   |

```shell
# List all runtime images (fetches from GitHub)
csghub-airgap list-images

# List only inference images
csghub-airgap list-images --category inference

# List images from a specific csghub-server version
csghub-airgap list-images --version v1.2.0

# List GPU images across all categories
csghub-airgap list-images --compute gpu

# List CPU images for finetune and inference
csghub-airgap list-images --category finetune --category inference --compute cpu

# List images from a local csghub-server directory (air-gapped)
csghub-airgap list-images \
  --source local \
  --local-path /path/to/csghub-server \
  --category inference

# Output as JSON for scripting
csghub-airgap list-images --category notebook --format json
```

## The --skip Flag

`--skip` accepts comma-separated, repeatable values (case-insensitive). Two kinds of values:

**Kinds (coarse-grained)**

| Value     | create behavior                              | deploy behavior                               |
| :-------- | :------------------------------------------- | :-------------------------------------------- |
| `k3s`     | Skip K3s download                            | Skip K3s install                              |
| `charts`  | Skip chart pull (reuse existing files), still export main images | Skip chart install/upgrade, still import images |
| `image`   | Skip main chart image export                 | Skip main chart image import                  |
| `runtime` | Skip runtime image export                    | Skip runtime image import                     |
| `package` | Skip NVIDIA package download                 | Skip NVIDIA package install                   |
| `tar`     | Skip packing: no `.tar`, keep the plain directory | Skip unpacking: `--input` must point at an already-unpacked directory |

**Chart names**

| Value      | Description                                                       |
| :--------- | :----------------------------------------------------------------- |
| `csghub`   | Skip the csghub chart (pull, its images, runtime images)          |
| `longhorn` | Skip the longhorn chart                                            |
| `nvdp`     | Skip the NVIDIA device plugin chart (and package operations)      |

> `--skip image` and `--skip runtime` are independent: `image` controls main chart images, `runtime` controls runtime images. Use both to skip all images.
> Chart names not present in the bundle are ignored with a warning.

**Image-only mode**: `--skip k3s,charts,package` (without `--skip image`) enters "export images only" mode and shows a main image selection UI per chart. Adding `--quiet` skips the UI and exports all main images directly.

```shell
# Export only images, skip the main image selection (export all main images)
sudo csghub-airgap create --skip k3s,charts,package --quiet
```

## Complete Workflow Examples

### Example 1: Basic Air-Gap Deployment

```shell
# 1) On the internet-connected machine (produces /opt/csghub-bundle.tar)
sudo csghub-airgap create --output /opt/csghub-bundle

# 2) Copy the archive to the air-gapped machine (USB, SCP, etc.)
scp /opt/csghub-bundle.tar user@airgap-host:/root/

# 3) Deploy on the air-gapped machine (auto-unpacked)
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle.tar \
  --domain csghub.internal.company.com
```

### Example 2: GPU Environment

```shell
# Bundle (includes nvdp and NVIDIA packages)
sudo csghub-airgap create --output /opt/csghub-gpu

# Deploy (auto-installs nvdp when a GPU is detected; auto-disables without one)
sudo csghub-airgap deploy \
  --input /opt/csghub-gpu \
  --domain csghub.gpu.company.com
```

### Example 3: Export/Import Images Only (Reuse Existing K3s)

```shell
# On the bundle machine
sudo csghub-airgap create --output /opt/csghub-img --skip k3s,charts,package

# On the target machine (K3s already installed, import images only)
sudo csghub-airgap deploy --input /opt/csghub-img --skip k3s,charts
```

### Example 4: Redeploy

```shell
sudo csghub-airgap cleanup --yes
sudo csghub-airgap deploy --input /opt/csghub-bundle --domain csghub.company.com
```

## Post-Deployment Verification

```shell
# Check CSGHub pods are ready
kubectl get pods -n csghub

# List namespaces
kubectl get ns
```

## Troubleshooting

### Permission Denied

Most commands require root privileges. Always run with `sudo`:

```shell
sudo csghub-airgap [command]
```

### Architecture Mismatch

Ensure the bundle architecture matches your system; re-create the bundle with the matching `--arch`:

```shell
uname -m
sudo csghub-airgap create --arch $(uname -m)
```

### Debug Mode

Enable debug logging for detailed output:

```shell
sudo csghub-airgap create --debug
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

### Common Error Messages

| Error                   | Solution                                                |
| :---------------------- | :------------------------------------------------------ |
| `permission denied`     | Run command with `sudo`                                 |
| `architecture mismatch` | Create bundle with correct `--arch` flag                |
| `bundle not found`      | Verify `--input` path is correct                        |
| `k3s already running`   | Run `csghub-airgap cleanup` first                       |
| stale image cache       | Re-create with `--force-pull` to force a fresh pull     |
| GitHub access restricted | Add `--proxy`; or use `list-images --source local`      |

## Security Notes

- This tool requires root privileges for deployment and cleanup operations
- Bundle files may contain sensitive configuration data - secure them appropriately
- Always verify bundle integrity before deployment in air-gapped environments
- The tool does not run any background daemons or services

## Uninstallation

To completely remove csghub-airgap:

```shell
# Remove the binary
sudo rm /usr/local/bin/csghub-airgap

# Remove any created bundles (optional)
sudo rm -rf /opt/csghub-* ./csghub-airgap*

# Clean up any deployments
sudo csghub-airgap cleanup --yes 2>/dev/null || true
```
