# License Header Templates

This document provides the standard license headers to use for files in the ObjectWeaver repository.

## Community Edition Files (AGPL-3)

For all files **outside** the `ee/` directory, use this header:

### Go Files
```go
// Copyright (c) 2025 ObjectWeaver
//
// This file is part of ObjectWeaver Community Edition.
// ObjectWeaver Community Edition is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE.txt or https://www.gnu.org/licenses/agpl-3.0.html
```

### JavaScript/TypeScript Files
```javascript
// Copyright (c) 2025 ObjectWeaver
//
// This file is part of ObjectWeaver Community Edition.
// ObjectWeaver Community Edition is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE.txt or https://www.gnu.org/licenses/agpl-3.0.html
```

### Python Files
```python
# Copyright (c) 2025 ObjectWeaver
#
# This file is part of ObjectWeaver Community Edition.
# ObjectWeaver Community Edition is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
# See LICENSE.txt or https://www.gnu.org/licenses/agpl-3.0.html
```

## Enterprise Edition Files (ObjectWeaver Commercial License)

For all files **inside** the `ee/` directory, use this header:

### Go Files
```go
// Copyright (c) 2025 ObjectWeaver
//
// This file is part of ObjectWeaver Enterprise Edition.
// ObjectWeaver Enterprise Edition is licensed under the ObjectWeaver Commercial License.
// See ee/LICENSE for details.
```

### JavaScript/TypeScript Files
```javascript
// Copyright (c) 2025 ObjectWeaver
//
// This file is part of ObjectWeaver Enterprise Edition.
// ObjectWeaver Enterprise Edition is licensed under the ObjectWeaver Commercial License.
// See ee/LICENSE for details.
```

### Python Files
```python
# Copyright (c) 2025 ObjectWeaver
#
# This file is part of ObjectWeaver Enterprise Edition.
# ObjectWeaver Enterprise Edition is licensed under the ObjectWeaver Commercial License.
# See ee/LICENSE for details.
```

## Important Notes

1. **Default Licensing**: Files without a header in the root or non-ee directories are assumed to be AGPL-3 licensed
2. **Third-Party Code**: Maintain original license headers for any third-party code
3. **Generated Code**: Generated files (protobuf, etc.) should include both a generation notice and the appropriate license header
4. **Test Files**: Test files should have the same license header as the code they test

## Adding Headers Automatically

You can use a script to add headers to files. Example for Go files:

```bash
# Add AGPL-3 header to all Go files outside ee/
find . -name "*.go" -not -path "./ee/*" -not -path "./vendor/*" -exec sed -i '' '1i\
// Copyright (c) 2025 ObjectWeaver\
//\
// This file is part of ObjectWeaver Community Edition.\
// ObjectWeaver Community Edition is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).\
// See LICENSE.txt or https://www.gnu.org/licenses/agpl-3.0.html\
' {} \;

# Add ObjectWeaver header to all Go files inside ee/
find ./ee -name "*.go" -exec sed -i '' '1i\
// Copyright (c) 2025 ObjectWeaver\
//\
// This file is part of ObjectWeaver Enterprise Edition.\
// ObjectWeaver Enterprise Edition is licensed under the ObjectWeaver Commercial License.\
// See ee/LICENSE for details.\
' {} \;
```

Note: The above script is for reference. Please review and test before applying to your codebase.
