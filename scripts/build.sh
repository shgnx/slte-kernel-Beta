#!/usr/bin/env bash
#
# 构建 SLTE 内核产物（libclash.so）。
#
# 产物是应用侧 `kernel-core/src/main/jniLibs/<abi>/` 的输入，构建完需要连同
# `libclash.h`、SHA256SUMS 摘要一起放回去（见 README「产物交付」）。
#
# 用法：
#   ./scripts/build.sh              # 构建 + ABI 校验 + 输出到 dist/
#   ./scripts/verify.sh <file.so>   # 只校验一个已有产物（见 scripts/verify.sh）
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ---- 可覆盖的构建参数 -------------------------------------------------------

: "${SLTE_ABI:=arm64-v8a}"
: "${SLTE_API_LEVEL:=28}"

# 上游内核版本：写进 mihomo 的 constant.Version，会显示在 App 的「关于」页
: "${SLTE_MIHOMO_VERSION:=v1.19.30}"

# 构建标签：android 隐含 linux（Go 对 GOOS=android 的既有约定），
# native/platform 下的 `// +build linux` 文件因此才会被编入
: "${SLTE_GO_TAGS:=android cmfa with_gvisor}"

# 工具链必须与既有产物一致，否则等于换了一个内核运行时
: "${SLTE_GO_TOOLCHAIN:=go1.26.4}"
: "${SLTE_NDK_VERSION:=28.2.13676358}"

: "${ANDROID_SDK_ROOT:=${ANDROID_HOME:-$HOME/Library/Android/sdk}}"

: "${SLTE_OUT:=$ROOT/dist}"

# 产物文件名。必须与应用仓库 kernels.json 里该内核的 `artifact` 完全一致：
# Go 的 c-shared 产物没有 SONAME，应用侧 C 桥记录下来的依赖名就是链接时的文件名，
# 名字对不上会在运行期报 "library not found"。
: "${SLTE_ARTIFACT:=libclash.so}"

# ---------------------------------------------------------------------------

case "$SLTE_ABI" in
    arm64-v8a)
        GOARCH_TARGET=arm64
        NDK_TRIPLE=aarch64-linux-android
        ;;
    armeabi-v7a)
        GOARCH_TARGET=arm
        NDK_TRIPLE=armv7a-linux-androideabi
        ;;
    *)
        echo "不支持的 ABI：${SLTE_ABI}（可选 arm64-v8a / armeabi-v7a）" >&2
        exit 1
        ;;
esac

if [[ -z "${NDK_HOME:-}" ]]; then
    for candidate in "$ANDROID_SDK_ROOT/ndk/$SLTE_NDK_VERSION" "$ANDROID_SDK_ROOT/ndk-bundle"; do
        if [[ -d "$candidate" ]]; then NDK_HOME="$candidate"; break; fi
    done
fi

if [[ -z "${NDK_HOME:-}" || ! -d "$NDK_HOME" ]]; then
    echo "找不到 NDK ${SLTE_NDK_VERSION}。请设置 NDK_HOME，或确认 ANDROID_SDK_ROOT=$ANDROID_SDK_ROOT" >&2
    exit 1
fi

case "$(uname -s)" in
    Darwin) NDK_HOST_TAG=darwin-x86_64 ;;
    Linux)  NDK_HOST_TAG=linux-x86_64 ;;
    *)      echo "不支持的构建主机：$(uname -s)" >&2; exit 1 ;;
esac

NDK_BIN="$NDK_HOME/toolchains/llvm/prebuilt/$NDK_HOST_TAG/bin"
CLANG="$NDK_BIN/${NDK_TRIPLE}${SLTE_API_LEVEL}-clang"

if [[ ! -x "$CLANG" ]]; then
    echo "找不到 NDK 交叉编译器：$CLANG" >&2
    exit 1
fi

mkdir -p "$SLTE_OUT"
TARGET="$SLTE_OUT/$SLTE_ARTIFACT"

echo "==> 工具链"
echo "    Go        : $(GOTOOLCHAIN=$SLTE_GO_TOOLCHAIN go version)"
echo "    CC        : $CLANG"
echo "    tags      : $SLTE_GO_TAGS"
echo "    mihomo    : $SLTE_MIHOMO_VERSION"
echo "    产物名    : $SLTE_ARTIFACT"
echo "    输出      : $TARGET"

cd "$ROOT"

CC="$CLANG" \
CXX="${CLANG}++" \
CGO_ENABLED=1 \
GOOS=android \
GOARCH="$GOARCH_TARGET" \
GOTOOLCHAIN="$SLTE_GO_TOOLCHAIN" \
go build \
    -tags "$SLTE_GO_TAGS" \
    -buildmode=c-shared \
    -trimpath \
    -ldflags "-X github.com/metacubex/mihomo/constant.Version=$SLTE_MIHOMO_VERSION -w -s -buildid=" \
    -o "$TARGET" \
    ./native

"$ROOT/scripts/verify.sh" "$TARGET"

echo
echo "==> 产物"
echo "    $TARGET"
echo "    ${TARGET%.so}.h"
echo "    sha256: $(shasum -a 256 "$TARGET" | cut -d' ' -f1)"
echo
echo "把它连同上面的 sha256 一起放回应用仓库的 kernel-core/src/main/jniLibs/${SLTE_ABI}/。"
