#!/usr/bin/env bash

# Script to sync go.mod dependencies from a go version -m output file
# Usage: ./sync-deps-from-binary.sh <reference-version-file> <target-binary> [go.mod-path]

set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <reference-version-file> <target-binary> [go.mod-path]"
    echo ""
    echo "  reference-version-file: File containing 'go version -m' output from reference binary"
    echo "  target-binary:          The plugin binary to compare against"
    echo "  go.mod-path:            Path to go.mod file (default: ./go.mod)"
    echo ""
    echo "This script will:"
    echo "  1. Read dependencies from the reference version file"
    echo "  2. Compare with dependencies in the target binary"
    echo "  3. Add replace directives only for mismatched versions"
    echo "  4. Run 'go mod tidy' to clean up"
    exit 1
fi

REFERENCE_VERSION_FILE="$1"
TARGET_BINARY="$2"
GOMOD_PATH="${3:-go.mod}"

if [ ! -f "$REFERENCE_VERSION_FILE" ]; then
    echo "Error: Reference version file '$REFERENCE_VERSION_FILE' not found"
    exit 1
fi

if [ ! -f "$TARGET_BINARY" ]; then
    echo "Error: Target binary '$TARGET_BINARY' not found"
    exit 1
fi

if [ ! -f "$GOMOD_PATH" ]; then
    echo "Error: go.mod file '$GOMOD_PATH' not found"
    exit 1
fi

GOMOD_DIR=$(dirname "$GOMOD_PATH")

echo "Reference version file: $REFERENCE_VERSION_FILE"
echo "Target binary: $TARGET_BINARY"
echo "Target go.mod: $GOMOD_PATH"
echo ""

# Extract dependencies from reference version file
echo "Parsing dependencies from reference version file..."
REFERENCE_DEPS=$(mktemp)
grep -E '^\s+dep\s+' "$REFERENCE_VERSION_FILE" | awk '{print $2 " " $3}' > "$REFERENCE_DEPS"

REFERENCE_COUNT=$(wc -l < "$REFERENCE_DEPS" | xargs)
echo "Found $REFERENCE_COUNT dependencies in reference"

# Extract dependencies from target binary
echo "Extracting dependencies from target binary..."
TARGET_DEPS=$(mktemp)
go version -m "$TARGET_BINARY" | grep -E '^\s+dep\s+' | awk '{print $2 " " $3}' > "$TARGET_DEPS"

TARGET_COUNT=$(wc -l < "$TARGET_DEPS" | xargs)
echo "Found $TARGET_COUNT dependencies in target"
echo ""

# Store reference versions
declare -A reference_versions
while read -r module version; do
    reference_versions["$module"]="$version"
done < "$REFERENCE_DEPS"

# Store target versions
declare -A target_versions
while read -r module version; do
    target_versions["$module"]="$version"
done < "$TARGET_DEPS"

rm -f "$REFERENCE_DEPS" "$TARGET_DEPS"

# Find mismatches
echo "Comparing dependency versions..."
declare -A mismatched_modules
MISMATCH_COUNT=0

# Debug: Check if logfmt is present
echo "DEBUG: Checking for logfmt..."
if [ -n "${reference_versions[github.com/go-logfmt/logfmt]}" ]; then
    echo "  Reference has logfmt: ${reference_versions[github.com/go-logfmt/logfmt]}"
else
    echo "  Reference does NOT have logfmt"
fi
if [ -n "${target_versions[github.com/go-logfmt/logfmt]}" ]; then
    echo "  Target has logfmt: ${target_versions[github.com/go-logfmt/logfmt]}"
else
    echo "  Target does NOT have logfmt"
fi
echo ""

for module in "${!target_versions[@]}"; do
    target_version="${target_versions[$module]}"
    reference_version="${reference_versions[$module]}"

    # Only add to mismatches if module exists in both and versions differ
    if [ -n "$reference_version" ] && [ "$reference_version" != "$target_version" ]; then
        echo "  Mismatch: $module"
        echo "    Reference: $reference_version"
        echo "    Target:    $target_version"
        mismatched_modules["$module"]="$reference_version"
        ((MISMATCH_COUNT++))
    fi
done

echo ""
if [ $MISMATCH_COUNT -eq 0 ]; then
    echo "No version mismatches found. Dependencies are already aligned."
    exit 0
fi

echo "Found $MISMATCH_COUNT mismatched dependencies"
echo ""

# Backup go.mod
BACKUP_FILE="${GOMOD_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$GOMOD_PATH" "$BACKUP_FILE"
echo "Created backup: $BACKUP_FILE"
echo ""

# Add replace directives only for mismatched modules
echo "Adding replace directives for mismatched dependencies..."
echo "" >> "$GOMOD_PATH"
echo "// Dependency version alignment - sync with reference binary" >> "$GOMOD_PATH"

for module in "${!mismatched_modules[@]}"; do
    version="${mismatched_modules[$module]}"
    echo "replace $module => $module $version" >> "$GOMOD_PATH"
    echo "  $module => $version"
done

echo ""
echo "Added $MISMATCH_COUNT replace directives"

echo ""
echo "Running go mod tidy..."
(cd "$GOMOD_DIR" && go mod tidy)

echo ""
echo "Successfully updated go.mod with replace directives for mismatched dependencies!"
echo ""
echo "Summary:"
echo "  Mismatched dependencies fixed: $MISMATCH_COUNT"
echo "  Backup saved to: $BACKUP_FILE"
echo ""
echo "To revert changes, run:"
echo "  mv $BACKUP_FILE $GOMOD_PATH"