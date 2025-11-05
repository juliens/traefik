#!/usr/bin/env bash

# Script to generate compile.go from a configuration file or command line arguments
# Usage:
#   ./generate-compile.sh <config-file> [output-file]
#   ./generate-compile.sh <repo@version> [repo@version...] [--output output-file]

set -e

# Check if first argument is a file
if [ -f "$1" ]; then
    # File mode
    CONFIG_FILE="$1"
    OUTPUT_FILE="${2:-cmd/plugindl/plugin.go}"
    USE_CONFIG_FILE=true
else
    # Command line mode
    USE_CONFIG_FILE=false
    OUTPUT_FILE="cmd/plugindl/plugin.go"

    # Check for --output flag
    ARGS=()
    for arg in "$@"; do
        if [ "$arg" == "--output" ]; then
            shift
            OUTPUT_FILE="$1"
            shift
        else
            ARGS+=("$arg")
        fi
    done

    if [ ${#ARGS[@]} -eq 0 ]; then
        echo "Error: No plugins specified"
        echo "Usage: $0 <repo@version> [repo@version...] [--output output-file]"
        echo "   or: $0 <config-file> [output-file]"
        exit 1
    fi
fi

# Read import paths from config file or command line
IMPORTS=()
VERSIONS=()

if [ "$USE_CONFIG_FILE" = true ]; then
    # Read from config file (skip empty lines and comments)
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        # Trim whitespace
        line=$(echo "$line" | xargs)
        # Remove surrounding quotes if present
        line=$(echo "$line" | sed 's/^"\(.*\)"$/\1/')
        # Skip null entries
        [[ "$line" == "null" ]] && continue
        # Skip if line is now empty after processing
        [[ -z "$line" ]] && continue

        # Check if line contains version specification (@x.x.x)
        if [[ "$line" =~ ^([^@]+)@(.+)$ ]]; then
            import_path="${BASH_REMATCH[1]}"
            version="${BASH_REMATCH[2]}"
            IMPORTS+=("$import_path")
            VERSIONS+=("$version")
        else
            IMPORTS+=("$line")
            VERSIONS+=("")
        fi
    done < "$CONFIG_FILE"
else
    # Read from command line arguments
    for arg in "${ARGS[@]}"; do
        # Check if arg contains version specification (@x.x.x)
        if [[ "$arg" =~ ^([^@]+)@(.+)$ ]]; then
            import_path="${BASH_REMATCH[1]}"
            version="${BASH_REMATCH[2]}"
            IMPORTS+=("$import_path")
            VERSIONS+=("$version")
        else
            IMPORTS+=("$arg")
            VERSIONS+=("")
        fi
    done
fi

if [ ${#IMPORTS[@]} -eq 0 ]; then
    echo "Error: No imports found in configuration file"
    exit 1
fi

# Generate the Go file
cat > "$OUTPUT_FILE" << 'EOF_HEADER'
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

EOF_HEADER

# Add imports with SHA256 aliases
for import_path in "${IMPORTS[@]}"; do
    # Calculate SHA256 hash of the import path
    alias_name=$(echo -n "$import_path" | shasum -a 256 | cut -d' ' -f1)
    echo "	p$alias_name \"$import_path\"" >> "$OUTPUT_FILE"
done

cat >> "$OUTPUT_FILE" << 'EOF_MIDDLE'
)

type plugin struct {
	Create any
	New    any
	Version string
}

func LoadPlugins() {
EOF_MIDDLE

# Add plugin registrations
for i in "${!IMPORTS[@]}"; do
    import_path="${IMPORTS[$i]}"
    version="${VERSIONS[$i]}"
    # Calculate SHA256 hash of the import path for the alias
    alias_name=$(echo -n "$import_path" | shasum -a 256 | cut -d' ' -f1)
    cat >> "$OUTPUT_FILE" << EOF
	pluginMap["$import_path"] = plugin{
		Create: p${alias_name}.CreateConfig,
		New:    p${alias_name}.New,
		Version: "${version}",
	}

EOF
done

cat >> "$OUTPUT_FILE" << 'EOF_FOOTER'
}

var pluginMap = map[string]plugin{}

func NewPlugin(ctx context.Context, name string, config string, next http.Handler) (http.Handler, error) {
  LoadPlugins()
	c := reflect.ValueOf(pluginMap[name].Create).Call([]reflect.Value{})[0].Interface()

	err := json.Unmarshal([]byte(config), &c)
	if err != nil {
		return nil, err
	}

	results := reflect.ValueOf(pluginMap[name].New).Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(next),
		reflect.ValueOf(c),
		reflect.ValueOf("name"),
	})

	var h http.Handler
	if !results[0].IsNil() {
		h = results[0].Interface().(http.Handler)
	}
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		if err != nil {
			return nil, err
		}
	}

	return h, nil
}

func Plugins() []string {
	var plugins []string
	for name, plugin := range pluginMap {
		plugins = append(plugins, name+"@"+plugin.Version)
	}

	return plugins
}

EOF_FOOTER

if [ "$USE_CONFIG_FILE" = true ]; then
    echo "Generated $OUTPUT_FILE from $CONFIG_FILE"
else
    echo "Generated $OUTPUT_FILE from command line arguments"
fi
echo "Plugins included: ${#IMPORTS[@]}"

# Update go.mod with specified versions
echo ""
echo "Updating go.mod with specified versions..."
FAILED_IMPORTS=()
SKIPPED_COUNT=0
for i in "${!IMPORTS[@]}"; do
    import_path="${IMPORTS[$i]}"
    version="${VERSIONS[$i]}"

    # Check if import already exists in go.mod
    existing_version=$(grep "^\s*$import_path " go.mod | awk '{print $2}')

    if [ -n "$existing_version" ]; then
        if [ -n "$version" ]; then
            # Version specified - check if it matches
            if [ "$existing_version" == "$version" ]; then
                echo "  - $import_path@$version (already in go.mod, skipping)"
                ((SKIPPED_COUNT++))
                continue
            else
                echo "  - $import_path@$version (updating from $existing_version)"
                if ! go get "$import_path@$version" 2>/dev/null; then
                    echo "    FAILED: Could not get $import_path@$version"
                    FAILED_IMPORTS+=("$import_path")
                fi
            fi
        else
            # No version specified - keep existing
            echo "  - $import_path (already in go.mod at $existing_version, skipping)"
            ((SKIPPED_COUNT++))
            continue
        fi
    else
        # Not in go.mod - need to add it
        if [ -n "$version" ]; then
            echo "  - $import_path@$version (adding)"
            if ! go get "$import_path@$version" 2>/dev/null; then
                echo "    FAILED: Could not get $import_path@$version"
                FAILED_IMPORTS+=("$import_path")
            fi
        else
            echo "  - $import_path (adding)"
            if ! go get "$import_path" 2>/dev/null; then
                echo "    FAILED: Could not get $import_path"
                FAILED_IMPORTS+=("$import_path")
            fi
        fi
    fi
done

echo ""
echo "Running go mod tidy..."
go mod tidy

# Comment out failed imports in the config file (only when using config file mode)
if [ ${#FAILED_IMPORTS[@]} -gt 0 ] && [ "$USE_CONFIG_FILE" = true ]; then
    echo ""
    echo "Commenting out failed imports in $CONFIG_FILE..."

    # Create a temporary file
    temp_file=$(mktemp)

    while IFS= read -r line || [ -n "$line" ]; do
        # Extract import path from line (removing version if present)
        clean_line=$(echo "$line" | xargs)
        import_in_line="${clean_line%%@*}"

        # Check if this import failed
        failed=false
        for failed_import in "${FAILED_IMPORTS[@]}"; do
            if [ "$import_in_line" == "$failed_import" ]; then
                failed=true
                break
            fi
        done

        # Comment out if failed, otherwise keep as is
        if [ "$failed" = true ] && [[ ! "$line" =~ ^[[:space:]]*# ]]; then
            echo "#$line" >> "$temp_file"
            echo "  - Commented out: $import_in_line"
        else
            echo "$line" >> "$temp_file"
        fi
    done < "$CONFIG_FILE"

    # Replace original file with updated version
    mv "$temp_file" "$CONFIG_FILE"
fi

echo ""
echo "Summary:"
echo "  Plugins processed: ${#IMPORTS[@]}"
echo "  Plugins skipped (already in go.mod): $SKIPPED_COUNT"
echo "  Plugins failed: ${#FAILED_IMPORTS[@]}"
echo "  Output file: $OUTPUT_FILE"
if [ ${#FAILED_IMPORTS[@]} -gt 0 ] && [ "$USE_CONFIG_FILE" = true ]; then
    echo "  Failed imports have been commented out in $CONFIG_FILE"
fi