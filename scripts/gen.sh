#!/usr/bin/env bash
# scripts/gen.sh — bash 版本，给 Git Bash / WSL / macOS / Linux 用
# 与 scripts/gen.ps1 行为等价。

set -euo pipefail

# 1. 切到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"
echo "[gen] project root: $PROJECT_ROOT"

# 2. 检查工具链
missing=()
for tool in protoc protoc-gen-go protoc-gen-go-grpc; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        missing+=("$tool")
    fi
done
if [ "${#missing[@]}" -ne 0 ]; then
    echo "[gen] missing: ${missing[*]}" >&2
    echo "      protoc           -> https://github.com/protocolbuffers/protobuf/releases"
    echo "      protoc-gen-go    -> go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo "      protoc-gen-grpc  -> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

# 3. 收集 proto 文件
mapfile -t protos < <(find api -type f -name '*.proto' | sort)
if [ "${#protos[@]}" -eq 0 ]; then
    echo "[gen] no .proto files under api/"
    exit 0
fi

echo "[gen] found ${#protos[@]} proto file(s):"
printf '       %s\n' "${protos[@]}"

# 4. 生成
# --proto_path=. 显式锁定 import 解析根为项目根，与 gen.ps1 行为对齐
protoc \
    --proto_path=. \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    "${protos[@]}"

echo "[gen] done."
