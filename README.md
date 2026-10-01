# SLTE Kernel

SLTE 的内核构建仓库。

应用仓库（`slte` / `slte-Beta`）**只保留预编译内核二进制**（`libclash.so` + 摘要 + 构建记录），
不再携带上游内核源码；需要重建或新增内核时在本仓库进行。

> **远端**：`beta` → https://github.com/shgnx/slte-kernel-Beta （当前用于测试与源码可得性）
> 稳定仓库 `slte-kernel` 创建后作为 `origin` 加入，两者按应用仓 `slte` / `slte-Beta` 的同款方式并行维护。

## 目录

| 路径 | 说明 |
|---|---|
| `native/` | 桥接层（Go 模块 `cfa`）：把 mihomo 暴露成应用侧的 C ABI |
| `third_party/mihomo/` | vendored mihomo v1.19.30 + 本地 outbound 补丁（1022 个文件） |
| `abis/libclash.symbols` | **冻结的 ABI 基线**：158 个导出符号 |
| `scripts/build.sh` | 构建 + ABI 校验 |
| `scripts/verify.sh` | 校验任意一个 `libclash.so` 是否满足 ABI |

## 构建

```bash
./scripts/build.sh
```

需要：Go（脚本会用 `GOTOOLCHAIN` 拉到指定版本）、Android NDK 28.2。
可用环境变量覆盖：`SLTE_ABI` / `SLTE_MIHOMO_VERSION` / `SLTE_GO_TOOLCHAIN` /
`SLTE_NDK_VERSION` / `NDK_HOME` / `SLTE_OUT`。

产物落在 `dist/`：`libclash.so` 与 `libclash.h`（cgo 生成的头）。

## 产物交付

把 `dist/libclash.so` 放回应用仓库：

```
slte/kernel-core/src/main/jniLibs/arm64-v8a/libclash.so
```

并同步三件事，缺一构建期会直接失败：

1. `SHA256SUMS` 里的摘要（`shasum -a 256 libclash.so`）；
2. `VERSION.md` 的构建记录（版本、工具链、变更原因）；
3. `kernel-core/kernels.json` 的元数据（新增内核时才需要）。

应用侧的 `:kernel-core:verifyNativeLibraries` 会做双向校验：清单声明的制品必须有摘要记录，
摘要记录里的条目也必须被清单声明。

## ABI 冻结规则

应用与内核是**跨仓库、二进制耦合**的：应用编译期看不到内核源码，签名不一致只会在运行期炸。
因此本仓库把导出符号集冻结成基线文件，构建时逐行比对：

- 新增/删除任何导出符号都会让 `build.sh` 失败；
- 确属有意的 ABI 变更时，需要**同时**更新 `abis/libclash.symbols`，并在应用仓库提升
  `kernel-core/kernels.json` 里的 `abiVersion`；
- cgo 生成的 `libclash.h` 必须与应用仓库 `kernel-core/src/main/cpp/libclash.h` 保持一致。

`native/bridge.h`（8 个回调函数指针 + 日志函数）是应用侧 C 桥与内核之间的另一半契约，
它与应用仓库的 `kernel-core/src/main/cpp/bridge.h` **必须保持同一份内容**。

## 新增第二个内核（如 mihomo-smart）

本仓库的定位是"每个内核族一条构建线"。新增同族内核时：

1. 在 `third_party/` 下引入该上游源码（或以补丁方式叠加到现有 mihomo 上）；
2. 复用 `native/`（同族内核的 C ABI 与配置格式一致）；
3. 用 `SLTE_GO_TAGS` / `SLTE_MIHOMO_VERSION` 或单独的脚本产出对应 `.so`；
4. 产物交付流程同上，并在应用仓库 `kernels.json` 里新增一条记录。

> 注意：不同 fork 的协议面可能不同。例如 `lux5am/mihomo-smart` 的 outbound 集合是本仓库的
> 子集（缺 jls / restls / shadowquic / shadowtls / zerotier），**直接换产物会丢协议**，
> 正确做法是把它的智能策略组代码叠加到本仓库的补丁线上重新编译。

## 许可证

本仓库包含的源码全部以 **GPL-3.0** 分发：

- mihomo：https://github.com/MetaCubeX/mihomo （见 `third_party/mihomo/LICENSE`）
- 桥接层派生自 ClashMetaForAndroid：https://github.com/MetaCubeX/ClashMetaForAndroid

GPL-3.0 要求分发二进制时提供对应源码。本仓库即对应源码：应用仓库随包分发的每个
`libclash.so`，其构建记录（`VERSION.md`）都指向本仓库的版本与构建命令。
