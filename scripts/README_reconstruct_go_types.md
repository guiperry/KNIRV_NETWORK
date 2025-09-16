# reconstruct_go_types.py — Enhanced Go Type Reconstruction Tool

## Purpose
Advanced tool for systematically restoring missing Go types and methods by parsing `go build` errors and intelligently reconstructing them from backup sources or usage patterns.

## Key Features

### 🔍 **Enhanced Error Detection**
- Detects undefined symbols, missing fields/methods, redeclared types
- Identifies method definition errors and circular dependency issues
- Handles "too many errors" scenarios gracefully
- Configurable error pattern filtering

### 🧠 **Intelligent Type Reconstruction**
- **Smart Type Inference**: Analyzes usage patterns and naming conventions
- **Method Extraction**: Automatically extracts and reconstructs associated methods
- **Dependency Resolution**: Uses topological sorting to determine optimal reconstruction order
- **Interface Generation**: Automatically generates interfaces for circular dependency resolution

### 🔄 **Iterative Reconstruction**
- **Auto-Convergence**: Repeatedly builds and fixes until compilation succeeds
- **Progress Tracking**: Detailed logging of each iteration's progress
- **Configurable Limits**: Set maximum iterations to prevent infinite loops

### 🛡️ **Enhanced Safety & Backup**
- **Comprehensive Backups**: Timestamped backups of entire `pkg/types` directory
- **Rollback Scripts**: Auto-generated bash scripts to undo all changes
- **Workspace Validation**: Pre-flight checks for Go modules and git repositories
- **Change Tracking**: Complete audit trail of all modifications

### ⚙️ **Configuration & Customization**
- **Config Files**: YAML/JSON configuration support
- **Type Mappings**: Custom type mappings and import preferences
- **Force Interfaces**: Configurable interface generation for specific types
- **Ignore Patterns**: Skip reconstruction for specified symbols

## Usage

### Basic Usage
```bash
# Dry-run (recommended first)
python3 scripts/reconstruct_go_types.py --backup-dir KNIRVORACLE_backup --workspace-root .

# Apply changes
python3 scripts/reconstruct_go_types.py --backup-dir KNIRVORACLE_backup --workspace-root . --apply
```

### Advanced Usage
```bash
# Iterative reconstruction (recommended for complex scenarios)
python3 scripts/reconstruct_go_types.py --backup-dir KNIRVORACLE_backup --iterative --verbose

# With custom configuration
python3 scripts/reconstruct_go_types.py --config reconstruction_config.yaml --iterative --apply

# Force interface generation for circular dependencies
python3 scripts/reconstruct_go_types.py --force-interfaces --apply

# Custom output directory
python3 scripts/reconstruct_go_types.py --output-dir pkg/types/generated --apply
```

### Rollback Operations
```bash
# Execute rollback using generated script
python3 scripts/reconstruct_go_types.py --rollback rollback_reconstruction.sh
```

## Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--backup-dir` | Directory containing backup Go files | `KNIRVORACLE_backup` |
| `--workspace-root` | Root directory of Go workspace | `.` |
| `--apply` | Actually write files (otherwise dry-run) | `False` |
| `--config` | Configuration file path (YAML/JSON) | `None` |
| `--iterative` | Use iterative reconstruction until build succeeds | `False` |
| `--max-iterations` | Maximum iterations for iterative mode | `5` |
| `--verbose, -v` | Enable verbose logging | `False` |
| `--force-interfaces` | Generate interfaces for circular dependencies | `False` |
| `--output-dir` | Output directory for reconstructed types | `pkg/types/reconstructed` |
| `--rollback` | Execute rollback using specified script | `None` |

## Configuration File Format

### YAML Example (`reconstruction_config.yaml`)
```yaml
# Custom type mappings
type_mappings:
  "DatabaseConnection": "interface{}"
  "Logger": "*log.Logger"

# Patterns to ignore during reconstruction
ignore_patterns:
  - "TestHelper"
  - "MockService"

# Types to always generate as interfaces
force_interfaces:
  - "ConsensusManager"
  - "P2PManager"

# Custom import mappings
custom_imports:
  "\.Database": "database/sql"
  "\.Context": "context"

# Maximum iterations for iterative mode
max_iterations: 10

# Output directory
output_dir: "pkg/types/reconstructed"
```

### JSON Example (`reconstruction_config.json`)
```json
{
  "type_mappings": {
    "DatabaseConnection": "interface{}",
    "Logger": "*log.Logger"
  },
  "ignore_patterns": ["TestHelper", "MockService"],
  "force_interfaces": ["ConsensusManager", "P2PManager"],
  "custom_imports": {
    "\\.Database": "database/sql",
    "\\.Context": "context"
  },
  "max_iterations": 10,
  "output_dir": "pkg/types/reconstructed"
}
```

## Workflow Recommendations

### 1. **Initial Assessment**
```bash
# Start with a dry-run to see what will be reconstructed
python3 scripts/reconstruct_go_types.py --verbose
```

### 2. **Iterative Reconstruction** (Recommended)
```bash
# Let the tool automatically iterate until build succeeds
python3 scripts/reconstruct_go_types.py --iterative --verbose --apply
```

### 3. **Manual Fine-tuning**
```bash
# Use configuration file for complex scenarios
python3 scripts/reconstruct_go_types.py --config config.yaml --iterative --apply
```

### 4. **Rollback if Needed**
```bash
# Use the auto-generated rollback script if issues arise
python3 scripts/reconstruct_go_types.py --rollback rollback_reconstruction.sh
```

## Output Structure

The tool creates reconstructed types in the specified output directory:
```
pkg/types/reconstructed/
├── ConsensusManager.go
├── P2PConsensusManager.go
├── MCPProcessor.go
└── ...
```

Each file includes:
- Proper package declaration
- Automatically generated imports
- Complete type definitions with methods
- JSON serialization tags where appropriate

## Logging and Monitoring

### Log Files
- **Location**: `reconstruct_go_types.log` in workspace root
- **Content**: Detailed reconstruction progress, errors, and decisions
- **Levels**: INFO, WARNING, ERROR, DEBUG (with `--verbose`)

### Progress Tracking
- Real-time console output showing current reconstruction step
- Iteration progress in iterative mode
- Detailed error analysis and resolution attempts

## Safety Features

### Automatic Backups
- **Comprehensive Backup**: Complete `pkg/types` directory backup before changes
- **Individual Backups**: `.bak` files for each modified file
- **Timestamped**: Backups include timestamp for easy identification

### Rollback Capability
- **Auto-generated Scripts**: Bash scripts to undo all changes
- **Complete Reversal**: Restores all modified files to original state
- **Verification**: Confirms successful rollback execution

### Validation
- **Pre-flight Checks**: Validates Go module structure and workspace
- **Dependency Analysis**: Detects potential circular dependencies
- **Build Verification**: Confirms changes don't break compilation

## Advanced Features

### Smart Type Inference
The tool analyzes your codebase to infer appropriate types:
- **Naming Conventions**: Fields ending in "Id" become `string`, "Count" become `int`
- **Usage Patterns**: Analyzes how fields are used to determine types
- **Method Signatures**: Extracts method signatures from backup files

### Dependency Resolution
- **Topological Sorting**: Reconstructs types in dependency order
- **Circular Detection**: Identifies and resolves circular dependencies
- **Interface Generation**: Automatically creates interfaces when needed

### Method Reconstruction
- **Complete Methods**: Extracts full method implementations from backups
- **Receiver Types**: Handles both pointer and value receivers
- **Method Grouping**: Associates methods with their parent types

## Troubleshooting

### Common Issues

1. **Build Still Fails After Reconstruction**
   - Use `--iterative` mode for automatic retry
   - Check logs for unresolved dependencies
   - Consider using `--force-interfaces` for circular dependencies

2. **Missing Methods**
   - Ensure backup files contain complete method definitions
   - Check that method signatures match expected usage
   - Use verbose mode to see method extraction details

3. **Import Errors**
   - Configure custom imports in config file
   - Check that generated imports are correct
   - Manually adjust imports in reconstructed files if needed

### Debug Mode
```bash
# Maximum verbosity for troubleshooting
python3 scripts/reconstruct_go_types.py --verbose --iterative
```

## Best Practices

1. **Always Start with Dry-run**: Review planned changes before applying
2. **Use Iterative Mode**: Let the tool automatically converge to a working state
3. **Configure Appropriately**: Use config files for complex reconstruction scenarios
4. **Review Generated Code**: Always review reconstructed types before committing
5. **Keep Backups**: The tool creates backups, but additional version control is recommended
6. **Test Thoroughly**: Run comprehensive tests after reconstruction

## Notes

- **Conservative Approach**: The tool prioritizes safety over completeness
- **Manual Review Required**: Always review generated code for correctness
- **Backup-Driven**: Best results when comprehensive backup files are available
- **Iterative Improvement**: Multiple runs may be needed for complex scenarios
- **Version Control Friendly**: All changes are tracked and reversible
