# CSGHub Airgap User Guide

English | [简体中文](docs/USER_GUIDE_cn.md)

CSGHub Airgap is a command-line tool for creating and deploying CSGHub offline bundles in air-gapped environments. Run `create` on a machine with internet access to build the bundle, then run `deploy` on the offline machine to install CSGHub.

```text
csghub-airgap <command> [options]
```

Available commands:

| Command       | Description                                          |
| :------------ | :--------------------------------------------------- |
| `create`      | Create an offline bundle on an internet-connected machine |
| `deploy`      | Deploy CSGHub from a bundle on an offline machine    |
| `cleanup`     | Remove K3s and all deployed services                 |
| `list-images` | List the available runtime images                    |

## Contents

- [create — Create a Bundle](#create--create-a-bundle)
- [deploy — Deploy Offline](#deploy--deploy-offline)
- [cleanup — Clean Up](#cleanup--clean-up)
- [list-images — List Runtime Images](#list-images--list-runtime-images)
- [The --skip Flag](#the---skip-flag)
- [Common Workflows](#common-workflows)
- [Post-Deployment Verification](#post-deployment-verification)
- [FAQ](#faq)
- [Notes](#notes)

---

## create — Create a Bundle

Run this on a machine with internet access. It packages every required component into the output directory.

```bash
sudo csghub-airgap create [options]
```

### Options

| Option           | Alias | Description                                                                                                                     | Default                      |
| :--------------- | :---- | :------------------------------------------------------------------------------------------------------------------------------ | :--------------------------- |
| `--output`       | `-o`  | Bundle output directory                                                                                                         | `./csghub-airgap-<date>`     |
| `--arch`         |       | Target architecture (`amd64`, `arm64`). Defaults to the host architecture                                                       | Current system architecture  |
| `--debug`        | `-d`  | Enable debug logging                                                                                                            | `false`                      |
| `--proxy`        |       | Prefix GitHub downloads with the `ghfast.top` proxy                                                                             | `false`                      |
| `--force-pull`   |       | Ignore local image caches (containerd cache + file cache) and always pull from the registry                                      | `false`                      |
| `--no-pkg-cache` |       | Ignore the NVIDIA package cache and always re-download apt/dnf/zypper packages                                                  | `false`                      |
| `--k3s-env`      |       | Extra environment variables passed to K3s (`KEY=VALUE`, repeatable)                                                              | `[]`                         |
| `--k3s-arg`      |       | Extra startup arguments passed to K3s (e.g. `--k3s-arg=--default-runtime=nvidia`)                                               | `[]`                         |
| `--skip`         |       | Skip components: chart names (`csghub`, `longhorn`, `nvdp`) and/or kinds (`k3s`, `charts`, `image`, `runtime`, `package`, `tar`). Comma-separated, repeatable |                              |
| `--quiet`        | `-q`  | Skip interactive prompts. Runtime image selection is skipped (no runtime images are selected)                                   | `false`                      |
| `--dry-run`      |       | Print the actions to be performed (downloads, image export, packing, etc.) without executing them                               | `false`                      |
| `--set`          | `-s`  | Set Helm values (e.g. `csghub.key=value`, repeatable)                                                                           | `[]`                         |

### Execution Flow

1. **Architecture check**: Verify the current platform can run the command (packing on macOS is supported; see Limitations below).
2. **NVIDIA package download**: Only when the target architecture matches the host, the `nvdp` chart is enabled, and `--skip package` is not set.
3. **K3s download**: Download the k3s binary, airgap images, install script, and helm binary (unless `--skip k3s`).
4. **Per-chart processing** (for each enabled chart):
   - Pull the Helm chart (unless `--skip charts`; if `--skip charts` is set but the chart file is missing locally, it is still pulled).
   - Export the chart's main images (unless `--skip image`).
   - For the `csghub` chart: show the runtime image selection UI, then export the selected runtime images (unless `--skip runtime` or `--quiet`).
5. **Write `metadata.yaml`** with the configuration used for this bundle.
6. **Pack**: Archive the entire output directory into `<dir>.tar` (unless `--skip tar`).

> **When runtime images are selected**: runtime image selection and export happen after the main csghub images have been exported, not at program startup.

### Runtime Image Selection UI

After the main csghub images are exported, the tool reads the runtime image configuration from csghub-server (by default from the `main` branch on GitHub) and presents a TUI:

```text
Select runtime images (space=toggle, a=all, A=none, enter=confirm, empty=skip)
```

- `Space` toggles the current item
- `a` selects all
- `A` clears the selection
- `Enter` confirms (selecting nothing skips runtime images)
- `q` / `Esc` cancels (also skips runtime images)

### Examples

```bash
# Basic bundle (interactive runtime image selection)
sudo csghub-airgap create

# Custom output directory with the proxy enabled
sudo csghub-airgap create \
  --output /opt/csghub-bundle \
  --proxy

# Build an arm64 bundle
sudo csghub-airgap create --arch arm64

# Skip runtime image selection entirely (non-interactive)
sudo csghub-airgap create --quiet

# Skip runtime images (the main image flow still runs, runtime export is skipped)
sudo csghub-airgap create --skip runtime

# Image-only mode: skip k3s / charts / packages (shows the main image selection UI)
sudo csghub-airgap create --skip k3s,charts,package

# Keep the plain directory (no .tar), useful with deploy --skip tar
sudo csghub-airgap create --skip tar

# Skip specific charts (csghub, longhorn, nvdp)
sudo csghub-airgap create --skip longhorn
sudo csghub-airgap create --skip longhorn,nvdp

# Force a fresh image pull (ignore local caches)
sudo csghub-airgap create --force-pull

# Custom Helm values and K3s arguments
sudo csghub-airgap create \
  --set csghub.global.gateway.external.domain=csghub.mycompany.com \
  --set longhorn.persistence.defaultClassReplicaCount=3 \
  --k3s-arg "--disable=servicelb"

# Extra K3s environment variables
sudo csghub-airgap create \
  --k3s-env "K3S_NODE_NAME=server01" \
  --k3s-env "K3S_TOKEN=mysecrettoken"
```

### Bundle Directory Structure

After `create` finishes, a sibling `csghub-airgap-<date>.tar` is produced containing the whole directory, making it easy to copy. For `deploy`, point `--input` at the archive (it is unpacked automatically) or at an already-unpacked directory (with `--skip tar`).

```text
csghub-airgap-<date>/
├── metadata.yaml                  # Bundle configuration (applied automatically on deploy)
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
├── csghub-runtime/                # Only created when runtime images are selected
│   └── images/csghub-runtime-<version>.tar
├── longhorn/
│   ├── charts/longhorn-<version>.tgz
│   └── images/longhorn-<version>.tar
└── nvidia-device-plugin/
    ├── charts/nvidia-device-plugin-<version>.tgz
    └── images/nvidia-device-plugin-<version>.tar
```

### Limitations

- **Cross-architecture packing**: When the target architecture differs from the host, NVIDIA packages are not downloaded. Download them manually on a Linux machine (e.g. `apt/dnf/zypper install nvidia-container-toolkit=...`).
- **Packing on macOS**: Supported, but NVIDIA packages cannot be downloaded on a non-Linux host and must be handled manually.
- **`--flannel-iface`**: Defaults to the interface on the packing machine (possibly `en0` on macOS). At deploy time, if it is not overridden with `--k3s-arg`, it is replaced with the target machine's default interface.

---

## deploy — Deploy Offline

Deploy CSGHub from a bundle on the offline machine.

```bash
sudo csghub-airgap deploy [options]
```

Root privileges are required.

### Options

| Option          | Alias | Description                                                                                              | Default                      |
| :-------------- | :---- | :------------------------------------------------------------------------------------------------------- | :--------------------------- |
| `--input`       | `-i`  | Bundle directory                                                                                         | `./csghub-airgap-<date>`     |
| `--debug`       | `-d`  | Enable debug logging                                                                                     | `false`                      |
| `--proxy`       |       | Prefix GitHub downloads with the `ghfast.top` proxy                                                      | `false`                      |
| `--k3s-env`     |       | Extra environment variables passed to K3s (`KEY=VALUE`, repeatable)                                      | `[]`                         |
| `--k3s-arg`     |       | Extra startup arguments passed to K3s                                                                    | `[]`                         |
| `--data`        | `-D`  | Data directory for K3s and Longhorn                                                                      | System default               |
| `--domain`      |       | CSGHub access domain                                                                                     | `csghub.example.com`         |
| `--quiet`       | `-q`  | Skip interactive prompts. multipathd is stopped and disabled automatically without asking                | `false`                      |
| `--dry-run`     |       | Print the actions to be performed (K3s install commands, chart installs, etc.) without executing them     | `false`                      |
| `--skip`        |       | Same as `create`: `k3s`, `charts`, `image`, `runtime`, `package`, `tar`, plus chart names                |                              |
| `--set`         | `-s`  | Set Helm values (e.g. `csghub.key=value`, repeatable)                                                    | `[]`                         |

### Execution Flow and Preflight

1. **System check**: Must be Linux; the bundle architecture must match the target machine, otherwise the command fails immediately.
2. **NVIDIA GPU detection**: If no NVIDIA GPU is present, the `nvdp` chart (and its package installation) is disabled automatically.
3. **NVIDIA package installation**: Installed when `nvdp` is enabled and `--skip package` is not set.
4. **Kernel parameters**: Write the required sysctl settings (skipped when `--skip k3s` is used with an external cluster).
5. **K3s installation**: Install k3s (unless `--skip k3s`).
6. **multipathd preflight**: If `multipathd` is running, prompt whether to stop and disable it (`--quiet` performs this automatically). Longhorn can conflict with multipath and cause PVC mounts to hang.
7. **Per-chart deployment** (for each enabled chart):
   - Import the main images into containerd (unless `--skip image`).
   - For the `csghub` chart: import runtime images (unless `--skip runtime`; only images selected during packing are imported).
   - Install or upgrade the chart (unless `--skip charts`).
8. **K3s configuration patch**: Apply domain and related settings (write `/etc/hosts` and patch the cluster CoreDNS). When k3s was not installed by this tool (`--skip k3s`, external cluster), the tool asks before applying; with `--quiet` it applies without asking.

### Examples

```bash
# Basic deployment
sudo csghub-airgap deploy --input /opt/csghub-bundle

# Custom domain and data directory
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --domain csghub.mycompany.com \
  --data /mnt/data

# Non-interactive deploy (auto-disables multipathd)
sudo csghub-airgap deploy --input /opt/csghub-bundle --quiet

# Skip K3s installation (only import images and install charts)
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip k3s

# Skip all image imports (images already loaded another way)
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip image

# Skip runtime image imports
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip runtime

# Skip specific charts
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip longhorn
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip nvdp

# Override Helm values at deploy time
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --set csghub.global.persistence.size=200Gi

# Extra K3s arguments and environment variables
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --k3s-arg "--default-runtime=nvidia" \
  --k3s-env "K3S_NODE_NAME=server01"
```

---

## cleanup — Clean Up

Uninstall K3s and remove all deployed services.

```bash
sudo csghub-airgap cleanup [options]
```

| Option  | Alias | Description                        | Default |
| :------ | :---- | :--------------------------------- | :------ |
| `--yes` | `-y`  | Skip confirmation and clean up now | `false` |

```bash
# Interactive cleanup (confirmation prompt + 10-second countdown, Ctrl+C to abort)
sudo csghub-airgap cleanup

# Clean up without confirmation
sudo csghub-airgap cleanup --yes
```

> **Warning**: `cleanup` removes the **entire machine's k3s installation and all its data** (not just CSGHub resources), including `/var/lib/rancher/k3s` (PVCs and Longhorn data), `/var/lib/kubelet`, the k3s binary, and the systemd service. **If k3s on this machine was installed by you rather than by this tool, it will be removed as well.** Double-check before proceeding.
>
> Interactive mode requires typing `yes` to confirm, followed by a 10-second countdown (`Ctrl+C` aborts).
> It runs `/usr/local/bin/k3s-uninstall.sh`; if that fails, run the script manually.

---

## list-images — List Runtime Images

Inspect the runtime images provided by csghub-server before building a bundle. No root required.

```bash
csghub-airgap list-images [options]
```

### Options

| Option         | Alias | Description                                                                                       | Default  |
| :------------- | :---- | :------------------------------------------------------------------------------------------------ | :------- |
| `--source`     |       | Configuration source: `github` (online) or `local` (local directory, for air-gapped use)          | `github` |
| `--version`    | `-v`  | csghub-server Git branch or tag (e.g. `main`, `v1.2.0`)                                           | `main`   |
| `--local-path` |       | Local directory containing csghub-server configuration (with `--source local`)                     | `.`      |
| `--category`   | `-c`  | Filter by category (repeatable): `evaluation`, `finetune`, `inference`, `job`, `notebook`, `space` | All      |
| `--compute`    |       | Filter by compute type (comma-separated): `cpu`, `gpu`, `dcu`                                     | All      |
| `--proxy`      |       | Prefix GitHub downloads with the `ghfast.top` proxy                                               | `false`  |
| `--format`     | `-f`  | Output format: `table` or `json`                                                                  | `table`  |

### Examples

```bash
# List all runtime images (configuration fetched from GitHub)
csghub-airgap list-images

# List inference images only
csghub-airgap list-images --category inference

# Multiple categories
csghub-airgap list-images --category finetune --category inference

# Filter by compute type
csghub-airgap list-images --compute gpu
csghub-airgap list-images --compute cpu,dcu

# Target a specific csghub-server version
csghub-airgap list-images --version v1.2.0

# Use the ghfast.top proxy
csghub-airgap list-images --proxy

# Read from a local configuration directory (air-gapped)
csghub-airgap list-images \
  --source local \
  --local-path /path/to/csghub-server \
  --category inference

# JSON output for scripting
csghub-airgap list-images --category notebook --format json
```

---

## The --skip Flag

`--skip` accepts comma-separated, repeatable values (case-insensitive). There are two kinds of values.

**Kinds (coarse-grained)**

| Value     | create behavior                                                  | deploy behavior                                                    |
| :-------- | :--------------------------------------------------------------- | :----------------------------------------------------------------- |
| `k3s`     | Skip the K3s download                                            | Skip the K3s install                                               |
| `charts`  | Skip chart pulls (reuse existing chart files), still export main images | Skip chart install/upgrade, still import images              |
| `image`   | Skip main image export                                           | Skip main image import                                             |
| `runtime` | Skip runtime image export                                        | Skip runtime image import                                          |
| `package` | Skip NVIDIA package download                                     | Skip NVIDIA package install                                        |
| `tar`     | Skip packing: no `.tar`, keep the plain directory                | Skip unpacking: `--input` must point at an already-unpacked directory |

**Chart names**

| Value      | Description                                                            |
| :--------- | :--------------------------------------------------------------------- |
| `csghub`   | Skip the csghub chart (pull, its images, and runtime images)           |
| `longhorn` | Skip the longhorn chart                                                |
| `nvdp`     | Skip the NVIDIA device plugin chart (and its package operations)       |

> `--skip image` and `--skip runtime` are independent: `image` controls main images, `runtime` controls runtime images. Use both to skip all images.
> Chart names that are not present in the bundle are ignored with a warning.

**Image-only mode**: `--skip k3s,charts,package` (without `--skip image`) enters "export images only" mode and shows a main image selection UI for each chart. Adding `--quiet` skips that UI and exports all main images directly.

```bash
# Export only images, skipping the main image selection (export all main images)
sudo csghub-airgap create --skip k3s,charts,package --quiet

# Export only runtime images (skip main images)
sudo csghub-airgap create --skip k3s,charts,package,tar,image
```

---

## Common Workflows

### Scenario 1: Basic Air-Gap Deployment

```bash
# 1) Build the bundle on the internet-connected machine (produces /opt/csghub-bundle.tar)
sudo csghub-airgap create --output /opt/csghub-bundle

# 2) Copy the archive to the air-gapped machine (USB, scp, etc.)
scp /opt/csghub-bundle.tar user@airgap-host:/root/

# 3) Deploy on the air-gapped machine (unpacked automatically)
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle.tar \
  --domain csghub.internal.company.com
```

### Scenario 2: GPU Environment

```bash
# Build the bundle (includes nvdp and NVIDIA packages)
sudo csghub-airgap create --output /opt/csghub-gpu

# Deploy (nvdp is installed automatically when a GPU is detected; disabled otherwise)
sudo csghub-airgap deploy \
  --input /opt/csghub-gpu \
  --domain csghub.gpu.company.com
```

### Scenario 3: Export/Import Images Only (Reuse Existing K3s)

```bash
# On the bundle machine
sudo csghub-airgap create --output /opt/csghub-img --skip k3s,charts,package

# On the target machine (K3s already installed, import images only)
sudo csghub-airgap deploy --input /opt/csghub-img --skip k3s,charts
```

### Scenario 4: Redeploy

```bash
sudo csghub-airgap cleanup --yes
sudo csghub-airgap deploy --input /opt/csghub-bundle --domain csghub.company.com
```

---

## Post-Deployment Verification

```bash
# Check whether the CSGHub pods are ready
kubectl get pods -n csghub

# List namespaces
kubectl get ns
```

---

## FAQ

| Symptom                          | Resolution                                                                       |
| :------------------------------- | :------------------------------------------------------------------------------- |
| `permission denied`              | Run with `sudo` (`deploy` / `cleanup` require root)                              |
| `architecture mismatch`          | Re-create the bundle with `--arch` matching the target machine                   |
| Stale image cache                | Re-create with `--force-pull` to pull fresh images                               |
| Want to skip runtime interaction | Add `--skip runtime` or `--quiet` when creating the bundle                       |
| Want to reuse existing charts    | Use `--skip charts` (missing files are pulled automatically)                     |
| GitHub access is restricted      | Add `--proxy`; or use `list-images --source local` with a local configuration directory |

---

## Notes

- Deployment and cleanup require root privileges; on Linux, `create` also requires root when downloading NVIDIA packages.
- The bundle configuration is stored in `metadata.yaml` inside the bundle and applied automatically on deploy. Command-line flags (`--set`, `--k3s-arg`, `--skip`, etc.) can override it at deploy time.
- **Image file cache**: Images pulled by `create` are cached in the user cache directory (`~/.cache/csghub-airgap/images` on Linux/macOS), keyed by image reference. If packing is interrupted and retried, already-pulled images are reused instead of downloaded again. The cache lives outside the bundle and is not transferred. Use `--force-pull` to refresh it.
- This tool does not run any background daemons.
