#!/bin/bash
# build-deb.sh — build a Debian package for EmilyOS
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERSION="${VERSION:-0.1.0}"
ARCH="amd64"
PKG="emilyos_${VERSION}_${ARCH}"
BUILD_DIR="${REPO_ROOT}/dist/${PKG}"

echo "Building EmilyOS ${VERSION} for ${ARCH}..."

# Build static binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o "${REPO_ROOT}/dist/emilyos" \
    "${REPO_ROOT}/cmd/emilyos"

echo "Binary: $(sha256sum "${REPO_ROOT}/dist/emilyos")"

# Create deb directory structure
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/lib/systemd/system"
mkdir -p "${BUILD_DIR}/etc/emilyos"
mkdir -p "${BUILD_DIR}/var/lib/emilyos/audit"

# Install binary
cp "${REPO_ROOT}/dist/emilyos" "${BUILD_DIR}/usr/bin/emilyos"
chmod 755 "${BUILD_DIR}/usr/bin/emilyos"

# Install Debian metadata
cp "${REPO_ROOT}/packaging/debian/control" "${BUILD_DIR}/DEBIAN/"
cp "${REPO_ROOT}/packaging/debian/postinst" "${BUILD_DIR}/DEBIAN/"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"

# Update version in control file
sed -i "s/^Version:.*/Version: ${VERSION}/" "${BUILD_DIR}/DEBIAN/control"

# Build the .deb
dpkg-deb --build "${BUILD_DIR}" "${REPO_ROOT}/dist/${PKG}.deb"
echo "Package: ${REPO_ROOT}/dist/${PKG}.deb"
echo "SHA256: $(sha256sum "${REPO_ROOT}/dist/${PKG}.deb")"
