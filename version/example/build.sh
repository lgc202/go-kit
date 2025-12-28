#!/bin/bash

# version 包构建脚本示例
# 此脚本演示如何在构建时注入版本信息

set -e

# 项目根目录
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# 版本信息包路径
VERSION_PKG="github.com/lgc202/go-kit/version"

# 获取版本信息
GIT_VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-unknown")
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_TREE_STATE="clean"
if [[ -n $(git status --porcelain 2>/dev/null) ]]; then
    GIT_TREE_STATE="dirty"
fi
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

# 构建 ldflags
LDFLAGS="-X ${VERSION_PKG}.gitVersion=${GIT_VERSION}"
LDFLAGS="${LDFLAGS} -X ${VERSION_PKG}.gitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -X ${VERSION_PKG}.gitTreeState=${GIT_TREE_STATE}"
LDFLAGS="${LDFLAGS} -X ${VERSION_PKG}.buildDate=${BUILD_DATE}"

# 输出构建信息
echo "Building with version info:"
echo "  Git Version:    ${GIT_VERSION}"
echo "  Git Commit:     ${GIT_COMMIT}"
echo "  Git Tree State: ${GIT_TREE_STATE}"
echo "  Build Date:     ${BUILD_DATE}"
echo ""

# 构建示例程序
OUTPUT="${ROOT_DIR}/bin/example"
echo "Building example to ${OUTPUT}..."
go build -ldflags "${LDFLAGS}" -o "${OUTPUT}" "${ROOT_DIR}/version/example"

echo ""
echo "Build complete! Run with:"
echo "  ${OUTPUT} -version"
echo "  ${OUTPUT} -version -json"
