#!/bin/bash

# Script to add copyright headers to all Go files in the repository
# Copyright (C) 2025-present ObjectWeaver.

# Define the copyright header
read -r -d '' HEADER << 'EOF'
// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.

EOF

# Counter for tracking
files_modified=0
files_skipped=0
total_files=0

echo "Starting to add copyright headers to Go files..."
echo "================================================"
echo ""

# Find all .go files recursively
while IFS= read -r -d '' file; do
    total_files=$((total_files + 1))
    
    # Check if file already has a copyright header
    if head -n 1 "$file" | grep -q "Copyright"; then
        echo "⏭️  Skipping (already has header): $file"
        files_skipped=$((files_skipped + 1))
    else
        # Create a temporary file with the header
        temp_file=$(mktemp)
        
        # Write header to temp file
        echo "$HEADER" > "$temp_file"
        
        # Append original file content
        cat "$file" >> "$temp_file"
        
        # Replace original file
        mv "$temp_file" "$file"
        
        echo "✅ Added header to: $file"
        files_modified=$((files_modified + 1))
    fi
done < <(find . -name "*.go" -type f -not -path "*/vendor/*" -not -path "*/.git/*" -print0)

echo ""
echo "================================================"
echo "Summary:"
echo "  Total Go files found: $total_files"
echo "  Files modified: $files_modified"
echo "  Files skipped: $files_skipped"
echo "================================================"
