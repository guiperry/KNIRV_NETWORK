# Contributing to KNIRV Client

Thank you for your interest in contributing to KNIRV Client! This document provides guidelines and information for contributors.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Architecture Principles](#architecture-principles)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)

---

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please:

- Be respectful and considerate
- Welcome newcomers and help them get started
- Focus on constructive feedback
- Respect differing viewpoints and experiences

---

## Getting Started

### Prerequisites

- **Go 1.23.3+** installed
- **Git** for version control
- **AI Provider API Key** (at least one): Cerebras, Gemini, Claude, OpenAI, or DeepSeek
- Familiarity with Go, AST parsing, and static analysis

### Setup Development Environment

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/knirvclient.git
   cd knirvclient
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your API keys
   ```

4. **Run the application**
   ```bash
   go run main.go
   ```

5. **Access the dashboard**
   ```
   http://localhost:3000
   ```

---

## Architecture Principles

KNIRV Client follows a **deterministic scanning architecture** with clear separation of concerns:

### 🎯 Core Principle: Determinism First

**Scanning and detection MUST be deterministic, reproducible, and work offline.**

```
┌─────────────────────────────────────────────────────────────┐
│  PHASE 1-4: DETERMINISTIC (NO AI)                           │
│  • Scanning (AST parsing, file analysis)                    │
│  • Detection (pattern matching, rule-based)                 │
│  • Data persistence                                         │
│  • Dashboard display                                        │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 5: AI-POWERED (USER-TRIGGERED ONLY)                  │
│  • User selects an issue                                    │
│  • AI generates solution                                    │
│  • User reviews and approves                                │
│  • Solution is applied                                      │
└─────────────────────────────────────────────────────────────┘
```

### ✅ DO: Deterministic Approaches

When contributing to scanning or detection:

- ✅ Use AST parsing for code analysis
- ✅ Use regex patterns for vulnerability detection
- ✅ Use cyclomatic complexity calculation for code quality
- ✅ Use CVE database lookups for dependency vulnerabilities
- ✅ Use static analysis for dead code detection
- ✅ Use rule-based pattern matching
- ✅ Ensure identical results for identical inputs
- ✅ Make features work offline (except external data sources)

### ❌ DON'T: Non-Deterministic Approaches

- ❌ Never use AI inference during scanning
- ❌ Never use AI inference during detection
- ❌ Never use AI for severity scoring
- ❌ Never use AI for risk prioritization
- ❌ Never make scanning dependent on external AI APIs
- ❌ Never introduce randomness in detection logic

### 🤖 AI Usage: Remediation Only

AI should **only** be used for:

- ✅ Generating fix recommendations (user-triggered)
- ✅ Explaining detected issues (user-triggered)
- ✅ Suggesting refactoring approaches (user-triggered)
- ✅ Analyzing breaking changes in dependencies (user-triggered)

AI should **never** be used for:

- ❌ Scanning code files
- ❌ Detecting vulnerabilities
- ❌ Identifying technical debt
- ❌ Scoring severity
- ❌ Building knowledge graphs
- ❌ Analyzing database schemas

---

## Development Workflow

### Branch Naming Convention

- `feature/description` - New features
- `fix/description` - Bug fixes
- `refactor/description` - Code refactoring
- `docs/description` - Documentation updates
- `test/description` - Test additions/improvements

### Commit Message Format

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `refactor` - Code refactoring
- `docs` - Documentation changes
- `test` - Test additions/improvements
- `perf` - Performance improvements
- `chore` - Maintenance tasks

**Examples:**
```
feat(scanner): add Python AST parsing support
fix(risk): correct SQL injection pattern matching
refactor(diagnoser): remove AI inference from detection
docs(readme): update architecture diagram
test(scanner): add unit tests for Go AST parser
```

---

## Coding Standards

### Go Code Style

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines:

1. **Use `gofmt`** to format all code
   ```bash
   gofmt -w .
   ```

2. **Use `golint`** to check code quality
   ```bash
   golint ./...
   ```

3. **Use `go vet`** to find suspicious code
   ```bash
   go vet ./...
   ```

### Code Organization

```
knirvclient/
├── main.go                    # Main server entry point
├── internal/
│   ├── scanner/              # Deterministic scanning (NO AI)
│   │   ├── scanner.go        # Main scanner logic
│   │   ├── ast_parser.go     # AST parsing
│   │   └── dependency.go     # Dependency extraction
│   ├── risk/                 # Deterministic detection (NO AI)
│   │   ├── diagnoser.go      # Main detection logic
│   │   ├── security.go       # Security pattern matching
│   │   ├── technical_debt.go # Code quality detection
│   │   └── obsolete.go       # Dead code detection
│   ├── remediation/          # AI-powered remediation (USER-TRIGGERED)
│   │   ├── service.go        # Remediation service
│   │   └── generator.go      # Solution generation
│   └── guardian/
│       └── guardian.go       # Orchestration
├── inference_engine/         # AI providers (REMEDIATION ONLY)
├── data_engine/              # Metrics and events
└── dashboard/                # Frontend UI
```

### Naming Conventions

- **Packages**: lowercase, single word (e.g., `scanner`, `risk`)
- **Files**: lowercase with underscores (e.g., `ast_parser.go`)
- **Functions**: camelCase for private, PascalCase for public
- **Variables**: camelCase
- **Constants**: PascalCase or UPPER_SNAKE_CASE

### Error Handling

Always handle errors explicitly:

```go
// ✅ Good
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to execute: %w", err)
}

// ❌ Bad
result, _ := someFunction()
```

### Logging

Use structured logging with appropriate levels:

```go
log.Printf("🔍 Scanning file: %s", filename)
log.Printf("✅ Scan completed: %d files processed", count)
log.Printf("⚠️  Warning: %s", warning)
log.Printf("❌ Error: %v", err)
```

---

## Testing Guidelines

### Unit Tests

All new code must include unit tests:

```go
func TestScanFile(t *testing.T) {
    scanner := NewScanner()
    result, err := scanner.ScanFile("testdata/sample.go")
    
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    
    if result.LinesOfCode != 100 {
        t.Errorf("Expected 100 lines, got %d", result.LinesOfCode)
    }
}
```

### Test Coverage

Maintain at least **80% test coverage** for new code:

```bash
go test -cover ./...
```

### Determinism Tests

For scanning and detection code, add determinism tests:

```go
func TestScanDeterminism(t *testing.T) {
    scanner := NewScanner()
    
    // Run scan twice
    result1, _ := scanner.ScanFile("testdata/sample.go")
    result2, _ := scanner.ScanFile("testdata/sample.go")
    
    // Results must be identical
    if !reflect.DeepEqual(result1, result2) {
        t.Error("Scan results are not deterministic")
    }
}
```

### Integration Tests

Test component interactions:

```go
func TestScanAndDetect(t *testing.T) {
    scanner := NewScanner()
    diagnoser := NewDiagnoser()
    
    // Scan code
    graph, _ := scanner.Scan("/path/to/project")
    
    // Detect issues
    issues, _ := diagnoser.Diagnose(graph)
    
    // Verify issues were detected
    if len(issues.SecurityVulns) == 0 {
        t.Error("Expected security vulnerabilities to be detected")
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestScanFile ./internal/scanner
```

---

## Pull Request Process

### Before Submitting

1. **Ensure all tests pass**
   ```bash
   go test ./...
   ```

2. **Format your code**
   ```bash
   gofmt -w .
   ```

3. **Check for issues**
   ```bash
   go vet ./...
   golint ./...
   ```

4. **Update documentation** if needed

5. **Add tests** for new functionality

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Architecture Compliance
- [ ] Scanning/detection code is deterministic (no AI)
- [ ] AI is only used for remediation (user-triggered)
- [ ] Code works offline (except external data sources)
- [ ] Results are reproducible

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] All tests pass
- [ ] Test coverage maintained/improved

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

### Review Process

1. **Automated checks** must pass (tests, linting)
2. **Code review** by at least one maintainer
3. **Architecture review** for compliance with determinism principles
4. **Testing verification** by reviewer
5. **Approval** and merge by maintainer

---

## Issue Reporting

### Bug Reports

Use the bug report template:

```markdown
**Describe the bug**
Clear description of the bug

**To Reproduce**
Steps to reproduce:
1. Go to '...'
2. Click on '...'
3. See error

**Expected behavior**
What you expected to happen

**Actual behavior**
What actually happened

**Environment**
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.23.3]
- KNIRV Client version: [e.g., 1.0.0]

**Logs**
Relevant log output
```

### Feature Requests

Use the feature request template:

```markdown
**Feature Description**
Clear description of the proposed feature

**Use Case**
Why is this feature needed?

**Proposed Solution**
How should this feature work?

**Architecture Compliance**
- [ ] Feature maintains deterministic scanning
- [ ] AI usage (if any) is user-triggered only
- [ ] Feature works offline (if applicable)

**Alternatives Considered**
Other approaches you've considered
```

### Security Vulnerabilities

**Do not** open public issues for security vulnerabilities. Instead:

1. Email: security@knirvclient.dev
2. Include detailed description
3. Include steps to reproduce
4. We will respond within 48 hours

---

## Component-Specific Guidelines

### Scanner Component (`internal/scanner/`)

**Purpose:** Deterministic code scanning and analysis

**Guidelines:**
- Use AST parsing for code analysis
- Extract dependencies via import statements
- Calculate metrics deterministically
- Never use AI inference
- Ensure reproducible results

**Example:**
```go
// ✅ Good: Deterministic AST parsing
func (s *Scanner) parseGoFile(path string) (*ast.File, error) {
    fset := token.NewFileSet()
    return parser.ParseFile(fset, path, nil, parser.ParseComments)
}

// ❌ Bad: AI-based analysis
func (s *Scanner) analyzeFile(path string) (string, error) {
    content, _ := os.ReadFile(path)
    return s.ai.Analyze(string(content)) // NEVER DO THIS
}
```

### Risk Diagnoser Component (`internal/risk/`)

**Purpose:** Deterministic issue detection

**Guidelines:**
- Use pattern matching for vulnerability detection
- Use cyclomatic complexity for code quality
- Use CVE databases for dependency risks
- Never use AI inference
- Ensure reproducible results

**Example:**
```go
// ✅ Good: Pattern-based SQL injection detection
func detectSQLInjection(code string) []Issue {
    pattern := regexp.MustCompile(`db\.Query\([^?]*\+`)
    matches := pattern.FindAllStringIndex(code, -1)
    // Convert matches to issues
}

// ❌ Bad: AI-based detection
func detectVulnerabilities(code string) []Issue {
    return ai.DetectIssues(code) // NEVER DO THIS
}
```

### AI Remediation Service (`internal/remediation/`)

**Purpose:** User-triggered solution generation

**Guidelines:**
- Only invoke AI when user requests a solution
- Provide context about the detected issue
- Generate actionable fix recommendations
- Allow user review before applying
- Log all AI interactions

**Example:**
```go
// ✅ Good: User-triggered remediation
func (r *RemediationService) GenerateSolution(issueID string, userID string) (*Solution, error) {
    issue := r.getIssue(issueID)
    context := r.buildContext(issue)
    solution := r.ai.GenerateFix(context)
    r.logAIInteraction(userID, issueID, solution)
    return solution, nil
}
```

---

## Documentation

### Code Comments

- Add comments for exported functions
- Explain complex logic
- Document assumptions
- Include examples for public APIs

```go
// ScanFile analyzes a single source code file and extracts metadata.
// It uses AST parsing to deterministically extract:
// - Lines of code
// - Cyclomatic complexity
// - Import dependencies
// - Function definitions
//
// Example:
//   result, err := scanner.ScanFile("/path/to/file.go")
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Printf("Lines: %d\n", result.LinesOfCode)
func (s *Scanner) ScanFile(path string) (*ScanResult, error) {
    // Implementation
}
```

### README Updates

Update README.md when:
- Adding new features
- Changing architecture
- Modifying API endpoints
- Updating dependencies

### API Documentation

Update `docs/API_DOCUMENTATION.md` when:
- Adding new endpoints
- Changing request/response formats
- Modifying authentication
- Adding query parameters

---

## Questions?

- **GitHub Discussions**: https://github.com/guiperry/knirvclient/discussions
- **GitHub Issues**: https://github.com/guiperry/knirvclient/issues
- **Email**: support@knirvclient.dev

---

## License

By contributing to KNIRV Client, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to KNIRV Client! 🚀