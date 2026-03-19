# csghub-airgap

`csghub-airgap` is a tool to generate offline bundles for deploying K3s and multiple charts (Csghub, Longhorn, Nvidia Device Plugin) with all required container images. It supports customizing chart options via CLI flags and `--set` style overrides, similar to Helm.

---

## Features

- Pull and package K3s binaries and charts for offline deployment.
- Export all required container images into a single archive.
- Support multiple charts with version control (`csghub`, `longhorn`, `nvdp`).
- CLI supports `--set` for overriding chart values without clearing defaults.
- Merge user-specified chart options safely using `With*Options`.
- Fully customizable architecture and output directory.

---

## Installation

Build from source:

```bash
git clone git@git-devops.opencsg.com:product/infra/deployment-automation/standard/csghub-airgap.git
cd csghub-airgap
go build -o csghub-airgap cmd/airgap/main.go