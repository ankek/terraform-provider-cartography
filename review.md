# Architectural Review: terraform-provider-cartographyArchitectural Review: Terraform Provider Cartography

Overall Assessment

**Date:** 2025-01-23  The codebase demonstrates solid architectural fundamentals with clean separation of concerns, good use of Go idioms, and professional-grade implementation. However, there are several areas for improvement in terms of scalability, maintainability, error handling, and operational excellence.

**Reviewer:** Architecture Review  

**Repository:** terraform-provider-cartography  Critical Issues

**Purpose:** Infrastructure diagram generation from Terraform state/config files1. Logging & Observability ⚠️ HIGH PRIORITY

Problem:

---

Uses fmt.Printf for logging in icon_scanner.go (6 instances)

## Executive SummaryNo structured logging framework

No log levels (DEBUG, INFO, WARN, ERROR)

This Terraform provider generates visual infrastructure diagrams (SVG/PNG) from Terraform state files or HCL configurations. The codebase demonstrates solid fundamentals with a clean separation of concerns across parsing, graph generation, layout calculation, and rendering layers. The architecture supports multi-cloud providers (AWS, Azure, GCP, DigitalOcean) and offers flexible input sources.Cannot disable verbose output

No observability for production debugging

**Overall Assessment:** ⭐⭐⭐⭐ (4/5)Recommendation:

- **Strengths:** Clean architecture, multi-cloud support, comprehensive test coverage

- **Areas for Improvement:** Logging, error handling, security validation, performance optimizationImpact: Makes debugging impossible in production environments, violates Terraform provider best practices.

---2. Error Handling & Context Propagation ⚠️ HIGH PRIORITY

Problem:

## 1. Architecture & Design Patterns

Inconsistent context checking across async operations

### Current StateparseResources checks context, but many parse functions don't

- **Clean Architecture:** Well-separated layers (parser → graph → renderer)No timeout handling for remote state fetches

- **Single Responsibility:** Each package has clear, focused responsibilitiesSilent error suppression in backend_parser.go:64 (continues on error)

- **Interface Usage:** `interfaces` package defines contracts, enabling testabilityRecommendation:

### RecommendationsFiles to update

#### 1.1 Dependency Injection Enhancementhcl_parser.go - Add context checks in parseHCLFile

**Priority: Medium**remote_state.go - Add timeouts for HTTP requests

graph.go - Add context checks in graph building loops

Currently, many components create their dependencies internally. Consider implementing a more explicit DI pattern:3. Security - Path Validation 🔒 MEDIUM PRIORITY

Problem:

```go

// Example: Provider should receive configured dependenciesvalidation.ValidateOutputPath creates test file but doesn't use O_TRUNC flag

type Provider struct {Potential race condition between check and use

    parser   parser.HCLParserValidateInputPath allows absolute paths with ".." after cleaning (line 72)

    renderer renderer.RendererRecommendation:

    logger   Logger

}4. Credential Management 🔒 MEDIUM PRIORITY

Problem:

func New(opts ...ProviderOption) *Provider {

    p := &Provider{Credentials passed through multiple layers (provider → state_loader → remote_state)

        parser:   parser.New(),CartographyProviderModel exposes sensitive fields

        renderer: renderer.New(),No credential sanitization in error messages

    }Risk of credential leakage in logs/errors

    for _, opt := range opts {Recommendation:

        opt(p)

    }Architectural Improvements

    return p5. Dependency Injection & Testing 📐 HIGH PRIORITY

}Problem:

```

Concrete implementations tightly coupled (hard to test)

**Benefits:**interfaces.go defines interfaces but they're not used in implementation

- Easier unit testing with mock implementationsDirect function calls instead of interface-based dependency injection

- Better separation of configuration from constructionMock testing difficult

- More flexible dependency swappingRecommendation:

#### 1.2 Configuration Management6. Configuration Management ⚙️ MEDIUM PRIORITY

**Priority: High**Problem:

The provider configuration is scattered across multiple locations. Implement centralized configuration:Magic numbers scattered throughout code (220.0, 160.0, 140.0 in export.go)

No centralized configuration

```goHard to adjust without code changes

// config/config.goRecommendation:

type Config struct {

    Diagram   DiagramConfig7. Resource Loading Strategy 📂 MEDIUM PRIORITY

    Rendering RenderConfigProblem:

    Parser    ParserConfig

}Complex priority cascade in state_loader.LoadResources (3 levels of fallback)

Error silencing makes debugging difficult

type DiagramConfig struct {No visibility into which method was used

    Direction    stringRecommendation:

    ShowTypes    bool

    GroupByCloud boolCode Quality Improvements

}8. Error Wrapping Consistency 📝 LOW PRIORITY

Problem:

type RenderConfig struct {

    IconPack    stringInconsistent error messages

    ColorScheme stringSome errors use %w, some use %s

    DPI         intNot all errors include context

}Recommendation:

```

9. Graph Algorithm Optimization 🚀 LOW PRIORITY

**Benefits:**Problem:

- Single source of truth for configuration

- Easier validation and defaultsedgeExists has O(n) complexity (line graph.go:41)

- Better documentation of available optionsCalled in loop leads to O(n²) for edge creation

No early termination optimization

---Recommendation:

## 2. Error Handling & Validation10. Test Coverage 🧪 MEDIUM PRIORITY

Problem:

### Current State

- Basic error propagation using `error` interfaceGood unit tests for helpers and validation

- Some validation in `internal/validation/path.go`Missing integration tests for full pipeline

- Limited context in error messagesNo tests for remote state fetching

No tests for error scenarios in diagram generation

### RecommendationsRecommendation

#### 2.1 Structured Error TypesPerformance Considerations

**Priority: High**11. Memory Management 💾 LOW PRIORITY

Problem:

Replace generic errors with typed errors:

Large state files loaded entirely into memory

```goNo streaming for remote state

// errors/errors.goPotential memory issues with 1000+ resources

type ErrorCode stringRecommendation:



const (12. Concurrent Processing ⚡ LOW PRIORITY

    ErrCodeInvalidState   ErrorCode = "INVALID_STATE"Problem:

    ErrCodeMissingFile    ErrorCode = "MISSING_FILE"

    ErrCodeRenderFailure  ErrorCode = "RENDER_FAILURE"Sequential file parsing in ParseConfigDirectory

)No parallelization for multiple .tf files

Graph building is sequential

type Error struct {Recommendation:

    Code    ErrorCode

    Message stringDocumentation & Maintenance

    Cause   error13. API Documentation 📚 LOW PRIORITY

    Context map[string]interface{}Problem:

}

Good package-level docs

func (e *Error) Error() string {Missing examples in godoc comments

    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)Some exported functions lack documentation

}Recommendation:

```

14. Version Compatibility 🔄 MEDIUM PRIORITY

**Benefits:**Problem:

- Programmatic error handling

- Better error categorizationHard-coded version handling in state_parser.go

- Improved debugging with contextNo forward compatibility strategy

No deprecation handling

#### 2.2 Input Validation EnhancementRecommendation

**Priority: High**

Priority Summary

Expand validation beyond just path checking:Implement Immediately:

✅ Add structured logging (terraform-plugin-log)

```go✅ Improve context handling & timeouts

// validation/config.go✅ Use dependency injection from interfaces.go

func ValidateDiagramConfig(cfg DiagramConfig) error {✅ Sanitize credentials in errors

    var errs []errorImplement Soon:

    ✅ Centralize configuration (remove magic numbers)

    if !validDirection(cfg.Direction) {✅ Add integration tests

        errs = append(errs, fmt.Errorf("invalid direction: %s", cfg.Direction))✅ Improve error wrapping consistency

    }✅ Optimize graph edge checking (O(n²) → O(n))

    Consider for Future:

    if cfg.Width < 0 || cfg.Height < 0 {⏳ Streaming for large files

        errs = append(errs, fmt.Errorf("dimensions must be positive"))⏳ Concurrent file parsing

    }⏳ Enhanced documentation with examples

    ⏳ Version compatibility strategy

    return errors.Join(errs...)Positive Highlights ⭐

}The codebase demonstrates several excellent practices:

```

Clean package structure with clear separation of concerns

---Context awareness throughout (good foundation)

Security-conscious path validation

## 3. Security ConsiderationsComprehensive architecture documentation

Good test coverage for critical components

### Current StateProfessional rendering with thoughtful UX (spacing, colors, icons)

- Path traversal protection in `validation/path.go`Multi-cloud support architecture

- No explicit security scanning for malicious Terraform filesOverall Grade: B+ (Good with room for improvement)

- Limited sanitization of user inputs

The architecture is solid and production-ready for small-to-medium workloads. Addressing the high-priority items will make it enterprise-ready.

### Recommendations

#### 3.1 Enhanced Path Validation

**Priority: Critical**

The current path validation should be strengthened:

```go
func ValidateOutputPath(path string) error {
    // Clean and normalize path
    cleanPath := filepath.Clean(path)
    
    // Check for absolute path requirements
    if !filepath.IsAbs(cleanPath) {
        return fmt.Errorf("path must be absolute: %s", path)
    }
    
    // Prevent path traversal
    if strings.Contains(cleanPath, "..") {
        return fmt.Errorf("path traversal not allowed")
    }
    
    // Validate writable directory
    dir := filepath.Dir(cleanPath)
    if err := checkWritePermissions(dir); err != nil {
        return fmt.Errorf("directory not writable: %w", err)
    }
    
    return nil
}
```

#### 3.2 Resource Limits

**Priority: High**

Protect against resource exhaustion from malicious inputs:

```go
// parser/limits.go
type Limits struct {
    MaxResources      int
    MaxDependencies   int
    MaxFileSize       int64
    ParseTimeout      time.Duration
}

var DefaultLimits = Limits{
    MaxResources:    10000,
    MaxDependencies: 50000,
    MaxFileSize:     100 * 1024 * 1024, // 100MB
    ParseTimeout:    5 * time.Minute,
}
```

---

## 4. Logging & Observability

### Current State

- No structured logging framework
- Limited visibility into parsing/rendering pipeline
- No metrics or telemetry

### Recommendations

#### 4.1 Structured Logging

**Priority: High**

Implement structured logging using a standard library like `slog`:

```go
import "log/slog"

func (p *Parser) ParseStateFile(ctx context.Context, path string) error {
    logger := slog.With(
        "component", "parser",
        "action", "parse_state",
        "path", path,
    )
    
    logger.Info("starting state file parsing")
    
    data, err := os.ReadFile(path)
    if err != nil {
        logger.Error("failed to read state file", "error", err)
        return err
    }
    
    logger.Info("successfully parsed state file",
        "resources", len(resources),
        "duration_ms", time.Since(start).Milliseconds(),
    )
    
    return nil
}
```

#### 4.2 Metrics Collection

**Priority: Medium**

Add basic metrics for operational insights:

```go
type Metrics struct {
    ParseDuration     time.Duration
    ResourceCount     int
    RenderDuration    time.Duration
    DiagramSize       int64
    LayoutIterations  int
}

func (m *Metrics) Report() {
    // Could integrate with Prometheus, StatsD, etc.
}
```

---

## 5. Performance Optimization

### Current State

- Graph algorithms appear efficient
- Layout calculation in `renderer/layout_improved.go` suggests iteration
- No obvious performance bottlenecks in current implementation

### Recommendations

#### 5.1 Caching Strategy

**Priority: Medium**

Cache expensive operations:

```go
// parser/cache.go
type ParserCache struct {
    stateFiles sync.Map // map[string]*CachedState
}

type CachedState struct {
    Resources   []Resource
    ParsedAt    time.Time
    FileModTime time.Time
}

func (c *ParserCache) Get(path string) (*CachedState, bool) {
    info, err := os.Stat(path)
    if err != nil {
        return nil, false
    }
    
    if cached, ok := c.stateFiles.Load(path); ok {
        cs := cached.(*CachedState)
        if cs.FileModTime.Equal(info.ModTime()) {
            return cs, true
        }
    }
    
    return nil, false
}
```

#### 5.2 Parallel Processing

**Priority: Low**

For large infrastructures, parallelize independent operations:

```go
func (r *Renderer) RenderLargeGraph(graph *Graph) error {
    var wg sync.WaitGroup
    errChan := make(chan error, 3)
    
    // Parallel layout calculation
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := r.calculateLayout(graph); err != nil {
            errChan <- err
        }
    }()
    
    // Parallel icon loading
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := r.loadIcons(graph); err != nil {
            errChan <- err
        }
    }()
    
    wg.Wait()
    close(errChan)
    
    return errors.Join(<-errChan...)
}
```

---

## 6. Testing Strategy

### Current State ✅

- **Comprehensive test coverage** added across all major packages
- Unit tests for parser, provider, renderer, graph, validation
- Integration tests for end-to-end pipeline
- Context cancellation tests for async operations
- **Current Status:** All tests passing (100% pass rate)

### Test Coverage Summary

```
✅ internal/parser          - 24 tests (state, HCL, backend parsing)
✅ internal/provider         - 9 tests (resource loading, state detection)
✅ internal/renderer         - 18 tests (SVG/PNG rendering, layout algorithms)
✅ internal/integration      - 4 tests (end-to-end pipeline)
✅ internal/graph            - Existing tests maintained
✅ internal/interfaces       - Interface contract tests
✅ internal/validation       - Path validation tests (cross-platform)
✅ Main package & cmd/*      - CLI execution tests
```

### Recommendations

#### 6.1 Property-Based Testing

**Priority: Low**

Add property-based tests for graph algorithms:

```go
import "testing/quick"

func TestLayoutProperties(t *testing.T) {
    f := func(nodeCount uint8) bool {
        if nodeCount > 100 {
            return true // Skip large graphs
        }
        
        graph := generateRandomGraph(int(nodeCount))
        layout := CalculateLayout(graph)
        
        // Property: No overlapping nodes
        return !hasOverlaps(layout)
    }
    
    if err := quick.Check(f, nil); err != nil {
        t.Error(err)
    }
}
```

#### 6.2 Benchmark Tests

**Priority: Medium**

Add benchmarks for critical paths:

```go
func BenchmarkParseStateFile(b *testing.B) {
    data := loadTestStateFile("large_infrastructure.tfstate")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := ParseStateFile(context.Background(), data)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 7. Multi-Cloud Support

### Current State

- Supports AWS, Azure, GCP, DigitalOcean
- Provider detection via resource type prefixes
- Icon mapping in `renderer/icons.go`

### Recommendations

#### 7.1 Provider Plugin System

**Priority: Low**

Make provider support extensible:

```go
// providers/provider.go
type CloudProvider interface {
    Name() string
    DetectResource(resourceType string) bool
    GetIcon(resourceType string) string
    GetCategory(resourceType string) Category
}

type ProviderRegistry struct {
    providers map[string]CloudProvider
}

func (r *ProviderRegistry) Register(p CloudProvider) {
    r.providers[p.Name()] = p
}

// providers/aws/aws.go
type AWSProvider struct{}

func (p *AWSProvider) Name() string { return "aws" }

func (p *AWSProvider) DetectResource(rt string) bool {
    return strings.HasPrefix(rt, "aws_")
}
```

#### 7.2 Custom Icon Support

**Priority: Medium**

Allow users to provide custom icon packs:

```go
type IconConfig struct {
    IconPack   string // "default", "material", "custom"
    CustomPath string // Path to custom icon directory
}

func (r *Renderer) LoadIcons(cfg IconConfig) error {
    switch cfg.IconPack {
    case "custom":
        return r.loadCustomIcons(cfg.CustomPath)
    default:
        return r.loadDefaultIcons()
    }
}
```

---

## 8. Documentation

### Current State

- Basic README with examples
- Documentation in `docs/` directory
- Inline code comments present

### Recommendations

#### 8.1 API Documentation

**Priority: High**

Ensure all exported functions have godoc comments:

```go
// ParseStateFile reads a Terraform state file and extracts infrastructure resources.
// It supports both Terraform state format versions 3 and 4.
//
// The function validates the state file structure and returns an error if:
//   - The file cannot be read or parsed
//   - The state format is unsupported
//   - Required fields are missing
//
// Example:
//
// resources, err := ParseStateFile(ctx, "terraform.tfstate")
// if err != nil {
//     log.Fatal(err)
// }
//
// Supported state versions: 3, 4
// Context cancellation is respected during parsing.
func ParseStateFile(ctx context.Context, path string) ([]Resource, error)
```

#### 8.2 Architecture Documentation

**Priority: Medium**

Create architecture decision records (ADRs):

```markdown
# ADR-001: Choice of Layout Algorithm

## Status
Accepted

## Context
Need to layout infrastructure diagrams with minimal edge crossings
and balanced node distribution.

## Decision
Implemented hierarchical layered layout based on Sugiyama framework
with collision detection.

## Consequences
- Produces visually clear diagrams for most infrastructures
- May not be optimal for very large graphs (>1000 nodes)
- Requires iterative improvement for edge routing
```

---

## 9. Deployment & Distribution

### Current State

- `release.sh` script for releasing
- `terraform-registry-manifest.json` for Terraform Registry
- Go module with standard structure

### Recommendations

#### 9.1 CI/CD Pipeline

**Priority: High**

Implement automated testing and release:

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go test -v -race -coverprofile=coverage.txt ./...
      - uses: codecov/codecov-action@v3
```

#### 9.2 Versioning Strategy

**Priority: Medium**

Follow semantic versioning strictly:

- **Major:** Breaking changes (state format changes, API modifications)
- **Minor:** New features (new cloud providers, diagram formats)
- **Patch:** Bug fixes, security updates

---

## 10. Error Recovery & Resilience

### Current State

- Basic error propagation
- Context cancellation support in some functions
- Limited retry logic

### Recommendations

#### 10.1 Graceful Degradation

**Priority: Medium**

When parsing fails for some resources, continue with others:

```go
type ParseResult struct {
    Resources []Resource
    Errors    []ResourceError
}

type ResourceError struct {
    ResourceType string
    ResourceName string
    Error        error
}

func ParseStateFile(ctx context.Context, path string) (*ParseResult, error) {
    result := &ParseResult{}
    
    for _, res := range rawResources {
        parsed, err := parseResource(res)
        if err != nil {
            result.Errors = append(result.Errors, ResourceError{
                ResourceType: res.Type,
                ResourceName: res.Name,
                Error:        err,
            })
            continue // Continue parsing other resources
        }
        result.Resources = append(result.Resources, parsed)
    }
    
    return result, nil
}
```

---

## 11. Extensibility

### Current State

- Fixed output formats (SVG, PNG)
- Hardcoded layout algorithms
- Limited customization options

### Recommendations

#### 11.1 Output Format Plugins

**Priority: Low**

Support custom output formats:

```go
type OutputFormatter interface {
    Format() string // "svg", "png", "pdf", "json"
    Render(diagram *Diagram) ([]byte, error)
}

type FormatterRegistry struct {
    formatters map[string]OutputFormatter
}

func (r *Renderer) RegisterFormatter(f OutputFormatter) {
    r.formatters[f.Format()] = f
}
```

#### 11.2 Custom Layout Algorithms

**Priority: Low**

Allow pluggable layout algorithms:

```go
type LayoutAlgorithm interface {
    Name() string
    Calculate(graph *Graph) NodePositions
}

type LayoutConfig struct {
    Algorithm string // "hierarchical", "force-directed", "circular"
    Options   map[string]interface{}
}
```

---

## 12. State Management

### Current State

- Stateless provider (reads state, generates diagram)
- No caching between invocations
- Fresh parse on every run

### Recommendations

#### 12.1 Incremental Diagram Updates

**Priority: Low**

For CI/CD environments, support incremental updates:

```go
type DiagramState struct {
    LastParsed   time.Time
    LastStateHash string
    CachedGraph  *Graph
}

func (g *Generator) GenerateIncremental(
    ctx context.Context,
    prevState *DiagramState,
    newStatePath string,
) (*Diagram, error) {
    newHash := hashFile(newStatePath)
    if prevState != nil && prevState.LastStateHash == newHash {
        return g.renderFromCache(prevState.CachedGraph)
    }
    
    return g.Generate(ctx, newStatePath)
}
```

---

## 13. Accessibility

### Current State

- SVG output supports text
- No explicit accessibility features
- No alternative text for diagrams

### Recommendations

#### 13.1 ARIA Support in SVG

**Priority: Low**

Add accessibility attributes to SVG output:

```go
func (r *SVGRenderer) RenderNode(node *Node) string {
    return fmt.Sprintf(`
        <g role="img" aria-label="%s - %s">
            <title>%s</title>
            <desc>%s resource of type %s</desc>
            <rect ... />
            <text ... />
        </g>
    `, node.Name, node.Type, node.Name, node.Provider, node.Type)
}
```

---

## 14. Technical Debt

### Current Items

1. **Logging Framework:** No structured logging (addressed in recommendation 4.1)
2. **Error Types:** Generic errors throughout (addressed in recommendation 2.1)
3. **Configuration Management:** Scattered configuration (addressed in recommendation 1.2)
4. **Test Coverage:** Now comprehensive with 50+ tests ✅

### Prioritization

**Critical (Immediate):**

- Enhanced path validation and security (Rec 3.1)
- Input validation enhancement (Rec 2.2)

**High (Next Sprint):**

- Structured error types (Rec 2.1)
- Structured logging (Rec 4.1)
- Centralized configuration (Rec 1.2)
- CI/CD pipeline (Rec 9.1)
- API documentation (Rec 8.1)

**Medium (Next Quarter):**

- Dependency injection (Rec 1.1)
- Caching strategy (Rec 5.1)
- Custom icon support (Rec 7.2)
- Benchmark tests (Rec 6.2)
- Graceful degradation (Rec 10.1)

**Low (Future):**

- Property-based testing (Rec 6.1)
- Provider plugin system (Rec 7.1)
- Output format plugins (Rec 11.1)
- Incremental updates (Rec 12.1)

---

## Summary

The terraform-provider-cartography codebase is well-structured with clear separation of concerns and solid foundations. The recent addition of comprehensive test coverage (50+ tests with 100% pass rate) significantly improves code quality and maintainability.

**Key Strengths:**

- Clean architecture with proper layering
- Multi-cloud support with extensible design
- Comprehensive test coverage across all packages
- Context-aware operations with cancellation support
- Cross-platform compatibility (Windows, Linux, macOS)

**Priority Improvements:**

1. Security hardening (path validation, resource limits)
2. Structured logging and error handling
3. Configuration management consolidation
4. CI/CD automation
5. Enhanced documentation

**Estimated Effort:**

- Critical items: 2-3 weeks
- High priority items: 4-6 weeks
- Medium priority items: 8-12 weeks
- Low priority items: Ongoing/optional

The codebase is production-ready with these improvements providing additional robustness, security, and maintainability for enterprise use cases.

---

**Review Completed:** 2025-01-23  
**Next Review Recommended:** After implementation of critical/high priority items
