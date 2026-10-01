#!/usr/bin/env bash
#
# 校验一个 libclash.so 是否满足冻结的内核 ABI。
#
# 用于两处：构建后自检；以及应用侧把远端下发/他人提供的产物装机前做一次独立核对。
#
# 用法：./scripts/verify.sh path/to/libclash.so
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASELINE="$ROOT/abis/libclash.symbols"

: "${SLTE_ABI:=arm64-v8a}"
: "${SLTE_NDK_VERSION:=28.2.13676358}"
: "${ANDROID_SDK_ROOT:=${ANDROID_HOME:-$HOME/Library/Android/sdk}}"

TARGET="${1:-}"
if [[ -z "$TARGET" || ! -f "$TARGET" ]]; then
    echo "用法：$0 path/to/libclash.so" >&2
    exit 2
fi

if [[ -z "${NDK_HOME:-}" ]]; then
    NDK_HOME="$ANDROID_SDK_ROOT/ndk/$SLTE_NDK_VERSION"
fi

case "$(uname -s)" in
    Darwin) NDK_HOST_TAG=darwin-x86_64 ;;
    Linux)  NDK_HOST_TAG=linux-x86_64 ;;
    *)      echo "不支持的校验主机：$(uname -s)" >&2; exit 1 ;;
esac

LLVM_BIN="$NDK_HOME/toolchains/llvm/prebuilt/$NDK_HOST_TAG/bin"
NM="$LLVM_BIN/llvm-nm"
READELF="$LLVM_BIN/llvm-readelf"

[[ -x "$NM" ]] || { echo "找不到 llvm-nm：$NM" >&2; exit 1; }

fail=0

# 1) 架构必须是目标 ABI
machine="$("$READELF" -h "$TARGET" | awk '/Machine:/ {print $2}')"
case "$SLTE_ABI:$machine" in
    arm64-v8a:AArch64|armeabi-v7a:ARM) ;;
    *) echo "✗ 架构不匹配：期望 ${SLTE_ABI}，实际 $machine" >&2; fail=1 ;;
esac

# 2) 导出符号集必须与基线完全一致（多一个、少一个都算 ABI 变更）
actual="$(mktemp)"
trap 'rm -f "$actual"' EXIT
"$NM" -D --defined-only "$TARGET" | awk '{print $3}' | sort > "$actual"

if ! diff -u "$BASELINE" "$actual" > /tmp/slte-kernel-abi.diff 2>&1; then
    echo "✗ 导出符号集与 ABI 基线不一致（差异见 /tmp/slte-kernel-abi.diff）：" >&2
    head -40 /tmp/slte-kernel-abi.diff >&2
    echo "  若确为有意的 ABI 变更，需要同时：更新 abis/libclash.symbols，" >&2
    echo "  并提升应用仓库 kernel-core/kernels.json 里的 abiVersion。" >&2
    fail=1
fi

# 3) 不能引入新的动态依赖（Go runtime 必须静态链入）
needed="$("$READELF" -d "$TARGET" | awk '/NEEDED/ {print $NF}' | tr -d '[]' | sort)"
expected=$'libc.so\nlibdl.so\nliblog.so'
if [[ "$needed" != "$expected" ]]; then
    echo "✗ 动态依赖与预期不符：" >&2
    diff <(printf '%s\n' "$expected") <(printf '%s\n' "$needed") >&2 || true
    fail=1
fi

if [[ $fail -ne 0 ]]; then
    exit 1
fi

echo "✓ ABI 校验通过：${SLTE_ABI}，$(wc -l < "$BASELINE" | tr -d ' ') 个导出符号，依赖 libc/libdl/liblog"
