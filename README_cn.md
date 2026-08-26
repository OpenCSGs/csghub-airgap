# CSGHub Airgap

一个用于在离线环境中创建和部署 CSGHub 离线包的 CLI 工具。在有网的机器上用 `create` 打包，在离线的机器上用 `deploy` 部署。该工具以命令行方式运行，无需任何运行时守护进程或后台服务。

## 功能特性

- **创建** 包含 K3s、Helm Chart 和容器镜像的离线包
- **部署** 在无互联网访问的环境下从离线包部署 CSGHub
- **清理** 按需移除 K3s 及所有服务
- **多架构** 支持（amd64、arm64）
- **选择性组件** 通过统一的 `--skip` 灵活跳过（图表、镜像、runtime 镜像、软件包、tar、K3s）
- **交互式 runtime 镜像选择**（TUI）在 `create` 时进行
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

## 命令

| 命令           | 说明                                                |
| :------------- | :-------------------------------------------------- |
| `create`       | 在有网机器上创建离线包（bundle）                    |
| `deploy`       | 在离线机器上从离线包部署 CSGHub                     |
| `cleanup`      | 清理 K3s 及所有已部署服务                           |
| `list-images`  | 查询可用的 runtime 镜像列表                         |

### 创建离线包 (create)

创建包含所有必要组件的离线包：

```shell
sudo csghub-airgap create [options]
```

#### 参数

| 参数             | 别名 | 说明                                                                   | 默认值                      |
| :--------------- | :--- | :--------------------------------------------------------------------- | :-------------------------- |
| `--output`       | `-o` | 离线包输出目录                                                          | `./csghub-airgap-<日期>`    |
| `--arch`         |      | 目标架构（`amd64`、`arm64`）                                           | 当前系统架构                |
| `--debug`        | `-d` | 开启调试日志                                                            | `false`                     |
| `--proxy`        |      | 对 GitHub 下载使用 `ghfast.top` 代理前缀                               | `false`                     |
| `--force-pull`   |      | 跳过本地镜像缓存（containerd 缓存 + 文件缓存），始终从仓库重新拉取镜像 | `false`                     |
| `--no-pkg-cache` |      | 跳过 NVIDIA 软件包缓存目录，始终重新下载 apt/dnf/zypper 软件包          | `false`                     |
| `--k3s-env`      |      | 传给 K3s 的额外环境变量（`KEY=VALUE`，可多次）                         | `[]`                        |
| `--k3s-arg`      |      | 传给 K3s 的额外启动参数（如 `--k3s-arg=--default-runtime=nvidia`）     | `[]`                        |
| `--skip`         |      | 跳过组件：图表名（`csghub`、`longhorn`、`nvdp`）和/或类别（`k3s`、`charts`、`image`、`runtime`、`package`、`tar`），逗号分隔、可多次 |              |
| `--quiet`        | `-q` | 跳过交互式提示；设置后 runtime 镜像选择被跳过（不选择任何 runtime 镜像）| `false`                     |
| `--set`          | `-s` | 设置 Helm values（如 `csghub.key=value`，可多次）                      | `[]`                        |

#### 执行流程

1. **架构检查**：确认当前平台可执行（macOS 上也可打包，见下方限制）。
2. **NVIDIA 软件包下载**：仅当目标架构与本机一致、启用了 `nvdp` 图表且未 `--skip package` 时执行。
3. **K3s 下载**：下载 k3s 二进制、airgap 镜像、安装脚本、helm 二进制（除非 `--skip k3s`）。
4. **逐图表处理**（每个启用的图表）：
   - 拉取 Helm chart（除非 `--skip charts`；若 `--skip charts` 但本地缺 chart 文件，仍会补拉）。
   - 导出该图表的主镜像（除非 `--skip image`）。
   - 若为 `csghub` 图表：弹出 runtime 镜像选择界面，确认后再导出选中的 runtime 镜像（除非 `--skip runtime`/`--skip image` 或 `--quiet`）。
5. **生成 `metadata.yaml`**，记录本次打包的配置。
6. **打包**：把整个输出目录打成 `<目录名>.tar`（除非 `--skip tar`）。

> **runtime 镜像选择时机**：runtime 镜像的选择和导出在 csghub 主镜像导出完成之后进行，而不是程序启动时。

#### runtime 镜像选择界面

在 csghub 主镜像导出后，程序会读取 csghub-server 的 runtime 镜像配置（默认从 GitHub 的 `main` 分支），并以 TUI 形式让你勾选：

```
Select runtime images (space=toggle, a=all, A=none, enter=confirm, empty=skip)
```

- `空格` 切换当前项选中状态
- `a` 全选
- `A` 全不选
- `回车` 确认（一个都不选 = 跳过，即不打包 runtime 镜像）
- `q` / `Esc` 取消（同样跳过 runtime 镜像）

#### 示例

```shell
# 基本打包（交互选择 runtime 镜像）
sudo csghub-airgap create

# 指定输出目录，并附带代理
sudo csghub-airgap create \
  --output /opt/csghub-bundle \
  --proxy

# 打 arm64 架构的包
sudo csghub-airgap create --arch arm64

# 不选择任何 runtime 镜像（无交互）
sudo csghub-airgap create --quiet

# 完全跳过 runtime 镜像
sudo csghub-airgap create --skip runtime

# 只导出镜像，跳过 k3s / 图表 / 软件包（image-only 模式，会弹出主镜像选择界面）
sudo csghub-airgap create --skip k3s,charts,package

# 不打包成 tar，保留纯目录（配合部署时 --skip tar）
sudo csghub-airgap create --skip tar

# 跳过指定图表
sudo csghub-airgap create --skip longhorn
sudo csghub-airgap create --skip longhorn,nvdp

# 强制重新拉取镜像（不使用本机缓存）
sudo csghub-airgap create --force-pull

# 自定义 Helm values 与 K3s 参数
sudo csghub-airgap create \
  --set csghub.global.gateway.external.domain=csghub.mycompany.com \
  --set longhorn.persistence.defaultClassReplicaCount=3 \
  --k3s-arg "--disable=servicelb"

# 追加 K3s 环境变量
sudo csghub-airgap create \
  --k3s-env "K3S_NODE_NAME=server01" \
  --k3s-env "K3S_TOKEN=mysecrettoken"
```

#### 离线包目录结构

`create` 完成后，除下面的目录外，还会在**同级**生成 `csghub-airgap-<日期>.tar`，即整个目录的打包文件，方便拷贝传输。`deploy` 时可直接把 `--input` 指向该文件，会自动解包；或指向已解包的目录（配合 `--skip tar`）。

```
csghub-airgap-<日期>/
├── metadata.yaml                  # 打包配置（部署时自动应用）
├── k3s/
│   ├── bin/k3s
│   ├── images/k3s-airgap-images-<arch>.tar.zst
│   ├── scripts/k3s-install.sh
│   ├── scripts/k3s-patch.sh
│   └── helm/helm-v3.20.1-linux-<arch>.tar.gz
├── packages/                      # NVIDIA container toolkit 软件包（Linux 上才有）
├── csghub/
│   ├── charts/csghub-<版本>.tgz
│   └── images/csghub-<版本>.tar
├── csghub-runtime/                # 仅当选中了 runtime 镜像时生成
│   └── images/csghub-runtime-<版本>.tar
├── longhorn/
│   ├── charts/longhorn-<版本>.tgz
│   └── images/longhorn-<版本>.tar
└── nvidia-device-plugin/
    ├── charts/nvidia-device-plugin-<版本>.tgz
    └── images/nvidia-device-plugin-<版本>.tar
```

#### 限制

- **跨架构打包**：目标架构与本机不一致时，不会下载 NVIDIA 软件包（需在 Linux 机器上手动下载，如 `apt/dnf/zypper install nvidia-container-toolkit=...`）。
- **macOS 上打包**：支持，但 NVIDIA 软件包无法在非 Linux 机器上下载，需手动处理。
- **`--flannel-iface`**：默认取自打包机器（macOS 上可能是 `en0`）；部署时若未用 `--k3s-arg` 覆盖，会自动替换为目标机器的默认网卡。

### 部署离线包 (deploy)

从离线包部署 CSGHub（需要 root 权限）：

```shell
sudo csghub-airgap deploy [options]
```

#### 参数

| 参数                | 别名 | 说明                                                               | 默认值                   |
| :------------------ | :--- | :----------------------------------------------------------------- | :----------------------- |
| `--input`           | `-i` | 离线包所在目录（或 `.tar` 打包文件）                                | `./csghub-airgap-<日期>` |
| `--debug`           | `-d` | 开启调试日志                                                        | `false`                  |
| `--proxy`           |      | 对 GitHub 下载使用 `ghfast.top` 代理前缀                           | `false`                  |
| `--k3s-env`         |      | 传给 K3s 的额外环境变量（`KEY=VALUE`，可多次）                     | `[]`                     |
| `--k3s-arg`         |      | 传给 K3s 的额外启动参数                                            | `[]`                     |
| `--data`            | `-D` | K3s 与 Longhorn 的数据目录                                          | 系统默认                 |
| `--domain`          |      | CSGHub 访问域名                                                     | `csghub.example.com`     |
| `--quiet`           | `-q` | 跳过交互式提示；设置后 multipathd 被自动停止并禁用（不再询问）       | `false`                  |
| `--skip`            |      | 同 `create`：`k3s`、`charts`、`image`、`runtime`、`package`、`tar` 及图表名 |                        |
| `--set`             | `-s` | 设置 Helm values（如 `csghub.key=value`，可多次）                  | `[]`                     |

#### 执行流程与预检

1. **系统检查**：必须是 Linux；离线包架构必须与目标机器一致，否则直接失败。
2. **NVIDIA GPU 检测**：本机无 NVIDIA GPU 时，自动禁用 `nvdp` 图表（及对应的软件包安装）。
3. **NVIDIA 软件包安装**：启用 `nvdp` 且未 `--skip package` 时安装。
4. **内核参数设置**：写入所需 sysctl 参数。
5. **K3s 安装**：安装 k3s（除非 `--skip k3s`）。
6. **multipathd 预检**：检测到 `multipathd` 运行时，提示是否停止并禁用（`--quiet` 下自动执行）。Longhorn 与 multipath 冲突可能导致 PVC 挂载卡住。
7. **逐图表部署**（每个启用的图表）：
   - 导入主镜像到 containerd（除非 `--skip image`）。
   - 对 `csghub` 图表：导入 runtime 镜像（除非 `--skip runtime`；仅当打包时选中了 runtime 镜像才导入）。
   - 安装或升级图表（除非 `--skip charts`）。
8. **K3s 配置修补**：应用域名等配置。

#### 示例

```shell
# 基本部署
sudo csghub-airgap deploy --input /opt/csghub-bundle

# 指定域名与数据目录
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --domain csghub.mycompany.com \
  --data /mnt/data

# 静默部署（自动禁用 multipathd，不交互）
sudo csghub-airgap deploy --input /opt/csghub-bundle --quiet

# 跳过 K3s 安装（仅导入镜像并安装图表）
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip k3s

# 跳过全部镜像导入（镜像已通过其他方式载入）
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip image

# 跳过 runtime 镜像导入
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip runtime

# 跳过指定图表
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip longhorn
sudo csghub-airgap deploy --input /opt/csghub-bundle --skip nvdp

# 部署时覆盖 Helm values
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --set csghub.global.persistence.size=200Gi

# 追加 K3s 启动参数与环境变量
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle \
  --k3s-arg "--default-runtime=nvidia" \
  --k3s-env "K3S_NODE_NAME=server01"
```

### 清理 (cleanup)

卸载 K3s 并清理所有已部署服务：

```shell
sudo csghub-airgap cleanup [options]
```

| 参数    | 别名 | 说明                         | 默认值  |
| :------ | :--- | :--------------------------- | :------ |
| `--yes` | `-y` | 跳过确认，直接执行清理       | `false` |

```shell
# 交互式清理（确认提示 + 10 秒倒计时缓冲，Ctrl+C 可取消）
sudo csghub-airgap cleanup

# 直接清理，不询问
sudo csghub-airgap cleanup --yes
```

> ⚠️ **注意**：cleanup 会删除**整台机器的 k3s 及其所有数据**（不只是 CSGHub 资源），包括 `/var/lib/rancher/k3s`（PVC、Longhorn 数据）、`/var/lib/kubelet`、k3s 二进制与 systemd 服务。**若该机器上的 k3s 是你自己安装的（而非本工具部署），也会一并删除**，请务必确认。
>
> 交互式模式下需输入 `yes` 确认，确认后有 10 秒倒计时缓冲（`Ctrl+C` 可取消）。实质是执行 `/usr/local/bin/k3s-uninstall.sh`。若失败可手动执行该脚本。

### 列出运行时镜像 (list-images)

在打包前查看 csghub-server 提供的 runtime 镜像（无需 root）：

```shell
csghub-airgap list-images [options]
```

| 参数           | 别名 | 说明                                                             | 默认值   |
| :------------- | :--- | :--------------------------------------------------------------- | :------- |
| `--source`     |      | 配置来源：`github`（在线）、`local`（本地目录，用于离线场景）     | `github` |
| `--version`    | `-v` | csghub-server 的 Git 分支或 tag（如 `main`、`v1.2.0`）           | `main`   |
| `--local-path` |      | 本地含 csghub-server 配置的目录（配合 `--source local` 使用）     | `.`      |
| `--category`   | `-c` | 按类别过滤（可多次）：`evaluation`、`finetune`、`inference`、`job`、`notebook`、`space` | 全部 |
| `--compute`    |      | 按计算类型过滤（逗号分隔）：`cpu`、`gpu`、`dcu`                  | 全部     |
| `--proxy`      |      | 对 GitHub 下载使用 `ghfast.top` 代理前缀                         | `false`  |
| `--format`     | `-f` | 输出格式：`table`、`json`                                        | `table`  |

```shell
# 列出全部 runtime 镜像（从 GitHub 拉取配置）
csghub-airgap list-images

# 只列 inference 类别
csghub-airgap list-images --category inference

# 指定 csghub-server 版本
csghub-airgap list-images --version v1.2.0

# 按计算类型过滤
csghub-airgap list-images --compute gpu
csghub-airgap list-images --compute cpu,dcu

# 从本地配置目录读取（离线场景）
csghub-airgap list-images \
  --source local \
  --local-path /path/to/csghub-server \
  --category inference

# JSON 输出，便于脚本处理
csghub-airgap list-images --category notebook --format json
```

## --skip 详解

`--skip` 接受逗号分隔、可多次指定的值（大小写不敏感）。取值分两类：

**类别（coarse-grained）**

| 值        | create 行为                                   | deploy 行为                                  |
| :-------- | :--------------------------------------------- | :------------------------------------------- |
| `k3s`     | 跳过 K3s 下载                                  | 跳过 K3s 安装                                |
| `charts`  | 跳过拉取图表（复用已有 chart 文件），仍导出主镜像 | 跳过图表安装/升级，仍导入镜像                |
| `image`   | 跳过所有镜像导出（主镜像 + runtime 镜像）       | 跳过所有镜像导入（主镜像 + runtime 镜像）     |
| `runtime` | 仅跳过 runtime 镜像导出                         | 仅跳过 runtime 镜像导入                       |
| `package` | 跳过 NVIDIA 软件包下载                          | 跳过 NVIDIA 软件包安装                        |
| `tar`     | 跳过打包：不生成 `.tar`，保留纯目录             | 跳过解包：`--input` 需指向已解包的目录        |

**图表名**

| 值        | 说明                                                              |
| :-------- | :---------------------------------------------------------------- |
| `csghub`  | 跳过 csghub 图表（拉取、其镜像、runtime 镜像）                    |
| `longhorn`| 跳过 longhorn 图表                                                |
| `nvdp`    | 跳过 NVIDIA 设备插件图表（并连带跳过软件包操作）                  |

> `--skip image` 会同时设置 `SkipRuntimeImages`，即一并跳过 runtime 镜像。
> 传入了图表清单中不存在的名字会被忽略并给出警告。

**image-only 模式**：`--skip k3s,charts,package`（且不 `--skip image`）会进入「只导出镜像」模式，为每个图表弹出主镜像选择界面。若同时加 `--quiet`，则跳过该选择界面，直接导出该图表全部主镜像。

```shell
# 只导出镜像，并跳过主镜像选择（导出全部主镜像）
sudo csghub-airgap create --skip k3s,charts,package --quiet
```

## 典型工作流

### 场景一：基本离线部署

```shell
# 1) 有网机器上打包（生成 /opt/csghub-bundle.tar）
sudo csghub-airgap create --output /opt/csghub-bundle

# 2) 拷贝打包文件到离线机器（U 盘 / scp 等）
scp /opt/csghub-bundle.tar user@airgap-host:/root/

# 3) 离线机器上部署（自动解包）
sudo csghub-airgap deploy \
  --input /opt/csghub-bundle.tar \
  --domain csghub.internal.company.com
```

### 场景二：GPU 环境

```shell
# 打包（包含 nvdp 与 NVIDIA 软件包）
sudo csghub-airgap create --output /opt/csghub-gpu

# 部署（本机检测到 GPU 时自动安装 nvdp；无 GPU 时自动禁用）
sudo csghub-airgap deploy \
  --input /opt/csghub-gpu \
  --domain csghub.gpu.company.com
```

### 场景三：仅导出/导入镜像（复用已有 K3s）

```shell
# 打包机器
sudo csghub-airgap create --output /opt/csghub-img --skip k3s,charts,package

# 目标机器（已有 K3s，只导入镜像）
sudo csghub-airgap deploy --input /opt/csghub-img --skip k3s,charts
```

### 场景四：重新部署

```shell
sudo csghub-airgap cleanup --yes
sudo csghub-airgap deploy --input /opt/csghub-bundle --domain csghub.company.com
```

## 部署后验证

```shell
# 查看 CSGHub 相关 Pod 是否就绪
kubectl get pods -n csghub

# 查看命名空间
kubectl get ns
```

## 常见问题

| 现象                    | 处理                                                                 |
| :---------------------- | :------------------------------------------------------------------- |
| `permission denied`     | 用 `sudo` 运行（deploy / cleanup 必须 root）                         |
| `architecture mismatch` | 用与目标机器一致的 `--arch` 重新打包                                  |
| 镜像缓存旧               | 重新打包时加 `--force-pull`，强制从仓库拉取                           |
| 想跳过 runtime 交互      | 打包时加 `--skip runtime` 或 `--quiet`                                |
| 想复用已有 chart 文件    | `--skip charts`（缺文件时会自动补拉）                                 |
| 网络访问 GitHub 受限     | 加 `--proxy`；或对 `list-images` 用 `--source local` 指向本地配置目录  |

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
