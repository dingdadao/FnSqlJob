#!/bin/bash
set -e

# 用法: ./release.sh [patch|minor|major]
# 默认: patch
BUMP=${1:-patch}

# 获取当前最新版本号，没有则从 v0.1.0 开始
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "当前版本: ${LATEST_TAG}"

# 解析版本号
VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# 递增版本号
case $BUMP in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
  *) echo "用法: ./release.sh [patch|minor|major]"; exit 1 ;;
esac

NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
echo "新版本: ${NEW_TAG}"

# 提交所有变更
git add -A
if git diff --cached --quiet; then
  echo "没有变更需要提交"
else
  git commit -m "release: ${NEW_TAG}"
fi

# 打标签
git tag -a "${NEW_TAG}" -m "Release ${NEW_TAG}"

# 构建
bash build.sh

echo ""
echo "=== Release ${NEW_TAG} 完成 ==="
echo "推送到远程: git push && git push --tags"
