# CSGHub Airgap

一个用于在离线环境中创建和部署 CSGHub 离线包的 CLI 工具。该工具以命令行方式运行，无需任何运行时守护进程或后台服务。

## 功能特性

- **创建** 包含 K3s、Helm Chart 和容器镜像的离线包
- **部署** 在无互联网访问的环境下从离线包部署 CSGHub
- **清理** 按需移除 K3s 及所有服务
- **多架构** 支持（amd64、arm64）
- **可配置组件**（Longhorn、NVDP 等）
- **列出运行时镜像** 支持按类别和计算类型过滤
- **无需守护进程** - 纯 CLI 工具，按需运行

## 前置要求

- Go 1.21 或更高版本（用于编译）
- Make（可选，用于使用 Makefile）
- Root/sudo 权限（用于部署和清理操作）

## 编译

### 编译当前系统版本

```bash
make build
```

### 编译 Linux 版本（amd64 + arm64）

```shell
make build-linux
```

或单独编译：

```shell
make build-linux-amd64
make build-linux-arm64
```

### 编译 macOS 版本（Intel + Apple Silicon）

```shell
make build-darwin
```

或单独编译：

```shell
make build-darwin-amd64
make build-darwin-arm64
```

### 编译所有平台

```shell
make build-all
```

### 清理编译产物

```shell
make clean
```

***注意：** 关于 Chart 的默认设置，请参考 `internal/manifest/charts`*

## 安装

赋予二进制文件可执行权限：

```shell
sudo chmod +x ./csghub-airgap-linux-amd64
```

### 命令

#### 创建离线包 (create)

创建包含所有必要组件的离线包：

```shell
sudo csghub-airgap create [options]
```

##### 全局选项

| 参数       | 简写 | 说明                            | 默认值                           |
| :--------- | :--- | :------------------------------ | :------------------------------- |
| `--output` | `-o` | 离线包输出目录                  | `./csghub-airgap-<日期>`         |
| `--arch`   | `-a` | 目标架构（amd64、arm64）        | 取决于本地环境                   |
| `--debug`  | `-d` | 启用调试日志                    | `false`                          |

***注意：** 交叉平台创建无法下载运行时软件包。*

##### K3s 选项

| 参数         | 说明                                    | 默认值  |
| :----------- | :-------------------------------------- | :------ |
| `--skip-k3s` | 跳过 K3s 安装                           | `false` |
| `--k3s-env`  | 为 K3s 设置额外的环境变量（KEY=VALUE）  | `[]`    |
| `--k3s-arg`  | 为 K3s 设置额外的 CLI 参数（例如 `--k3s-arg=--default-runtime=nvidia`） | `[]`    |

##### Chart 选项

| 参数                 | 简写  | 说明                                         | 默认值      |
| :------------------- | :---- | :------------------------------------------- | :---------- |
| `--csghub-version`   | `-cv` | CSGHub Chart 版本                            | （必需）    |
| `--csghub-runtime`   |       | 拉取 CSGHub 运行时镜像                       | `false`     |
| `--longhorn-version` | `-lv` | Longhorn Chart 版本                          | `1.11.1`    |
| `--skip-longhorn`    |       | 跳过 Longhorn 安装                           | `false`     |
| `--nvdp-version`     | `-nv` | NVIDIA 设备插件版本                          | `0.19.0`    |
| `--skip-nvdp`        |       | 跳过 NVDP 安装                               | `false`     |
| `--set`              | `-s`  | 设置 Helm 值（例如 csghub.key=value，可多次指定）| `[]`       |

##### 示例

```shell
# 基本离线包创建
sudo csghub-airgap create

# 创建 ARM64 架构的离线包并指定版本
sudo csghub-airgap create \
  --arch arm64 \
  --output /opt/csghub-bundle \
  --csghub-version 1.17.0 \
  --longhorn-version 1.11.0

# 跳过 Longhorn 和 NVDP
sudo csghub-airgap create \
  --skip-longhorn \
  --skip-nvdp \
  --output /tmp/bundle \
  --csghub-version 2.0.0

# 设置自定义 Helm 值
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --set csghub.global.gateway.external.domain=csghub.example.io \
  --set longhorn.csi.replicaCount=3

# 包含 CSGHub 运行时镜像
sudo csghub-airgap create \
  --csghub-runtime

# 添加 K3s 自定义参数
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --k3s-arg "--disable=traefik"

# 添加 K3s 环境变量
sudo csghub-airgap create \
  --csghub-version 2.0.0 \
  --k3s-env "K3S_TOKEN=mysecrettoken" \
  --k3s-env "K3S_DATASTORE_ENDPOINT=mysql://..."
```

***注意：** 你可以在 `csghub-airgap-<日期>/metadata.yaml` 中查看默认参数设置。部署时会自动应用这些参数，除非你手动指定覆盖。*

#### 部署离线包 (deploy)

从离线包部署 CSGHub：

```shell
sudo csghub-airgap deploy [options]
```

##### 选项

| 参数              | 简写   | 说明                                       | 默认值                    |
| :---------------- | :----- | :----------------------------------------- | :------------------------ |
| `--input`         | `-i`   | 离线包输入目录                             | `./csghub-airgap-<日期>`  |
| `--debug`         | `-d`   | 启用调试日志                               | `false`                   |
| `--skip-k3s`      |        | 跳过 K3s 安装                              | `false`                   |
| `--k3s-env`       |        | 为 K3s 设置额外的环境变量（KEY=VALUE）     | `[]`                      |
| `--k3s-arg`       |        | 为 K3s 设置额外的 CLI 参数                 | `[]`                      |
| `--data`          | `-D`   | 为 K3s 和 Longhorn 指定数据目录            | （系统默认）              |
| `--csghub-domain` | `-cd`  | 指定 CSGHub 域名                           | `csghub.example.com`      |
| `--skip-longhorn` |        | 跳过 Longhorn 安装                         | `false`                   |
| `--skip-nvdp`     |        | 跳过 NVDP 安装                             | `true`                    |
| `--set`           | `-s`   | 设置 Helm 值（例如 csghub.key=value，可多次指定） |                       |

##### 示例

```shell
# 基本部署
sudo csghub-airgap deploy --input /opt/csghub-bundle

# 使用自定义域名和数据目录部署
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --csghub-domain csghub.mycompany.com \
  --data /mnt/data  ## 同时用于 k3s 和 longhorn 卷

# 跳过 Longhorn 部署
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --skip-longhorn

# 启用 NVDP（默认禁用）
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --skip-nvdp=false  ## 这将自动在 --k3s-args 中添加 `--default-runtime=nvidia`

# 部署时设置自定义 Helm 值
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --set csghub.global.persistence.size=200Gi

# 添加 K3s 环境变量
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --k3s-env "K3S_NODE_NAME=server01" \
  --k3s-env "K3S_DATASTORE_CAFILE=/etc/ssl/certs/ca-certificates.crt"
```

#### 清理 (cleanup)

移除 K3s 及所有已部署的服务：

```shell
sudo csghub-airgap cleanup [options]
```

##### 选项

| 参数    | 简写 | 说明                         | 默认值  |
| :------ | :--- | :--------------------------- | :------ |
| `--yes` | `-y` | 跳过确认提示，直接执行清理   | `false` |

##### 示例

```shell
# 交互式清理（带确认提示）
sudo csghub-airgap cleanup

# 强制清理，无需确认
sudo csghub-airgap cleanup --yes
```

#### 列出运行时镜像 (list-images)

在创建离线包之前，列出 csghub-server 配置中的可用运行时镜像：

```shell
csghub-airgap list-images [options]
```

##### 选项

| 参数           | 简写 | 说明                                                     | 默认值     |
| :------------- | :--- | :------------------------------------------------------- | :--------- |
| `--source`     |      | 配置来源：`github`、`local`                               | `github`   |
| `--version`    | `-v` | csghub-server 的 Git 分支或标签（例如 `main`、`v1.2.0`）| `main`     |
| `--local-path` |      | 包含 csghub-server 配置的本地目录（与 `--source local` 配合使用） | `.`        |
| `--category`   | `-c` | 按类别过滤（可多次指定）：`evaluation`、`finetune`、`inference`、`job`、`notebook`、`space` |（全部）|
| `--compute`    |      | 按计算类型过滤（逗号分隔）：`cpu`、`gpu`、`dcu`          |（全部）   |
| `--format`     | `-f` | 输出格式：`table`、`json`                                | `table`    |

##### 示例

```shell
# 列出所有运行时镜像（从 GitHub 获取）
csghub-airgap list-images

# 仅列出推理镜像
csghub-airgap list-images --category inference

# 列出指定 csghub-server 版本的镜像
csghub-airgap list-images --version v1.2.0

# 列出所有类别中的 GPU 镜像
csghub-airgap list-images --compute gpu

# 列出微调和推理类别的 CPU 镜像
csghub-airgap list-images --category finetune --category inference --compute cpu

# 从本地 csghub-server 目录列出镜像（适用于离线环境）
csghub-airgap list-images \
  --source local \
  --local-path /path/to/csghub-server \
  --category inference

# 以 JSON 格式输出（便于脚本处理）
csghub-airgap list-images --category notebook --format json
```

## 完整工作流程示例

### 示例 1：基本离线部署

#### 在有互联网的机器上（创建离线包）

```shell
# 创建包含所有组件的离线包
sudo csghub-airgap create

# 验证离线包内容
ls -la ./csghub-airgap-2026-04-01
```

#### 传输到离线机器

```shell
# 通过 USB、SCP 等方式将离线包复制到离线机器
scp ./csghub-airgap-2026-04-01 user@airgap-machine:/root/
```

#### 在离线机器上（部署）

```shell
# 部署 CSGHub
sudo csghub-airgap deploy \
  --csghub-domain csghub.internal.company.com
```

### 示例 2：生产环境部署（自定义存储）

#### 创建离线包

```shell
sudo csghub-airgap create \
  --output /opt/csghub-prod-bundle \
  --csghub-version 2.0.0 \
  --longhorn-version 1.11.1 \
  --set csghub.global.persistence.storageClass=longhorn \
  --set csghub.global.persistence.size=200Gi
```

#### 部署

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-prod-bundle \
  --csghub-domain csghub.prod.company.com \
  --data /data
```

### 示例 3：GPU 加速部署

#### 创建包含 NVDP 的离线包

```shell
sudo csghub-airgap create \
  --output /opt/csghub-gpu-bundle \
  --csghub-version 2.0.0 \
  --nvdp-version 0.19.0
```

#### 部署 GPU 支持

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-gpu-bundle \
  --csghub-domain csghub.gpu.company.com \
  --skip-nvdp=false
```

### 示例 4：最小化部署（无 Longhorn、无 NVDP）

#### 创建离线包

```shell
sudo csghub-airgap create \
  --output /opt/csghub-minimal-bundle \
  --arch amd64 \
  --csghub-version 2.0.0 \
  --skip-longhorn \
  --skip-nvdp
```

#### 部署

```shell
sudo csghub-airgap deploy \
  --input /opt/csghub-minimal-bundle \
  --skip-longhorn \
  --csghub-domain csghub.minimal.company.com
```

### 示例 5：清理并重新部署

```shell
# 清理现有部署
sudo csghub-airgap cleanup --yes

# 重新部署
sudo csghub-airgap deploy \
  --input /opt/csghub-prod-bundle \
  --csghub-domain csghub.company.com
```

## 故障排除

### 权限被拒绝

大多数命令需要 root 权限。请始终使用 `sudo` 运行：

```shell
sudo csghub-airgap [command]
```

### 架构不匹配

确保离线包的架构与你的系统匹配：

```shell
# 检查系统架构
uname -m

# 使用正确的架构创建离线包
sudo csghub-airgap create --arch $(uname -m) --csghub-version 2.0.0
```

### 调试模式

启用调试日志以获取详细输出：

```shell
sudo csghub-airgap create --debug --csghub-version 2.0.0
sudo csghub-airgap deploy --debug --input /path/to/bundle
```

### 清理失败

如果清理失败，可以手动移除 K3s：

```shell
# 运行 K3s 卸载脚本
/usr/local/bin/k3s-uninstall.sh

# 或完全移除 K3s
/usr/local/bin/k3s-killall.sh
/usr/local/bin/k3s-uninstall.sh
```

### 创建离线包失败

检查磁盘空间和权限：

```shell
# 检查可用磁盘空间
df -h

# 检查输出目录的写入权限
ls -ld /path/to/output/directory

# 确保 Docker/容器运行时正在运行
systemctl status docker
```

### 部署失败

验证离线包完整性和系统要求：

```shell
# 检查离线包元数据
cat /path/to/bundle/metadata.yaml

# 确认 K3s 未在运行
systemctl status k3s

# 检查端口可用性
netstat -tulpn | grep -E '6443|8080'
```

### 常见错误信息

| 错误信息                 | 解决方法                                   |
| :----------------------- | :----------------------------------------- |
| `permission denied`      | 使用 `sudo` 运行命令                       |
| `architecture mismatch`  | 使用正确的 `--arch` 参数创建离线包         |
| `bundle not found`       | 确认 `--input` 路径是否正确                |
| `k3s already running`    | 先运行 `csghub-airgap cleanup`             |
| `disk space full`        | 释放磁盘空间或指定不同的 `--output` 目录   |

## 安全说明

- 该工具需要 root 权限来执行部署和清理操作
- 离线包文件可能包含敏感配置数据，请妥善保管
- 在离线环境中部署前，请始终验证离线包的完整性
- 该工具不会运行任何后台守护进程或服务

## 卸载

要完全移除 csghub-airgap：

```shell
# 移除二进制文件
sudo rm /usr/local/bin/csghub-airgap

# 移除已创建的离线包（可选）
sudo rm -rf /opt/csghub-* ./csghub-airgap*

# 清理所有已部署的服务
sudo csghub-airgap cleanup --yes 2>/dev/null || true
```
