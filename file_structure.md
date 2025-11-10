
## File Structure Reference

```
Source Code Location: /terraform-provider-cartography/internal/
├── provider/
│   ├── provider.go                 ← Provider configuration
│   ├── diagram_data_source.go      ← Data source (read-only)
│   ├── diagram_resource.go         ← Resource (CRUD)
│   ├── diagram_generator.go        ← Shared generation logic
│   └── state_loader.go             ← Smart loading orchestration
├── parser/
│   ├── state_parser.go             ← Terraform state parsing
│   ├── backend_parser.go           ← Backend detection
│   ├── remote_state.go             ← Remote state fetching
│   ├── hcl_parser.go               ← HCL file parsing
│   └── types.go                    ← Resource type definitions
├── graph/
│   └── graph.go                    ← Dependency graph building
├── renderer/
│   └── *.go                        ← SVG/PNG rendering
└── validation/
    └── path.go                     ← Path validation
```