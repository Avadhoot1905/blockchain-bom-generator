# SmartBOM — Comprehensive Engineering Audit Report

**Date:** 2026-06-10
**Scope:** Full codebase at commit `24e0cbb`
**Auditor:** Principal Security Architect / Supply-Chain Analysis

---

## PHASE 1: Repository Structure Analysis

### Top-Level Layout

```
blockchain-bom-generator/
├── cmd/smartbom/
│   ├── main.go                    # Entry point: delegates to cmd.Execute()
│   └── cmd/
│       ├── root.go                # Cobra root command, slog initialisation
│       ├── scan.go                # PRIMARY: the full 7-step analysis pipeline
│       ├── graph_cmd.go           # STUB: returns error "not yet implemented"
│       └── vuln_cmd.go            # STUB: returns error "scanners are stubs"
├── internal/
│   ├── model/
│   │   ├── component.go           # Domain constants + Component struct (ORPHANED)
│   │   └── dependency.go          # Dependency struct (ORPHANED)
│   ├── discovery/
│   │   ├── scanner.go             # FileScanner: walk → classify by extension
│   │   └── scanner_test.go
│   ├── git/
│   │   ├── manager.go             # go-git backed shallow clone / cleanup
│   │   └── manager_test.go
│   ├── parser/
│   │   ├── interface.go           # ParsedFile, Contract, Function, Parser interface
│   │   └── solidity/
│   │       ├── parser.go          # Regex-based Solidity parser (NO AST)
│   │       └── parser_test.go
│   ├── graph/
│   │   ├── graph.go               # Node, Edge, Graph + traversal methods
│   │   ├── graph_test.go
│   │   ├── builder.go             # Two-pass graph construction from ParsedFiles
│   │   └── builder_test.go
│   ├── semantic/
│   │   ├── interface.go           # Analyzer interface + Pipeline + DefaultPipeline()
│   │   ├── helpers.go             # inheritanceList(), functionList(), importList()
│   │   ├── token.go               # TokenAnalyzer: ERC20 / ERC721 / ERC1155
│   │   ├── proxy.go               # ProxyAnalyzer: UUPS / Transparent / Beacon
│   │   ├── oracle.go              # OracleAnalyzer: Chainlink / TWAP / Band
│   │   ├── governance.go          # GovernanceAnalyzer: Governor / DAO / Timelock
│   │   ├── treasury.go            # TreasuryAnalyzer: Vault / Multisig / Escrow
│   │   └── analyzer_test.go
│   ├── cyclonedx/
│   │   ├── builder.go             # graph.Graph → *cdx.BOM (CycloneDX 1.6 JSON)
│   │   └── builder_test.go
│   └── vuln/
│       └── scanner.go             # Scanner interface + Engine + 5 stub scanners
├── testdata/contracts/            # 5 Solidity fixture files
├── output/bom.json                # Sample output (scanned OpenZeppelin monorepo)
└── pkg/                           # Empty
```

**Go module:** `github.com/smartbom/smartbom`, Go 1.21

**Dependencies of note:**
- `github.com/CycloneDX/cyclonedx-go v0.9.0` — official CycloneDX encoder
- `github.com/go-git/go-git/v5 v5.11.0` — pure-Go git operations
- `github.com/google/uuid v1.6.0` — BOM serial number generation
- `github.com/spf13/cobra v1.8.0` — CLI framework

No on-chain RPC libraries. No AST libraries. No cryptographic analysis libraries. No vulnerability database clients.

---

### Architectural Overview

The system is a **linear, single-pass analysis pipeline** with no feedback loops, no persistence layer, and no graph database. Everything is in-memory for the duration of one invocation.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          cmd/smartbom scan                               │
└──────────────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────┐     ┌──────────────────┐
│  git.Manager    │────▶│  Repository on   │
│  CloneWithRef() │     │  local disk      │
└─────────────────┘     └────────┬─────────┘
                                 │ path
                                 ▼
                        ┌─────────────────────┐
                        │ discovery.FileScanner│
                        │ Scan(rootPath)       │
                        └────────┬────────────┘
                                 │ Project{SolidityFiles, VyperFiles, ...}
                                 ▼
                        ┌─────────────────────┐
                        │ solidity.Parser      │  (Vyper/Rust/Move: STUBS)
                        │ Parse(path) → x N   │
                        └────────┬────────────┘
                                 │ []*ParsedFile
                                 ▼
                        ┌─────────────────────┐
                        │ graph.Builder        │
                        │ Build(files)         │
                        └────────┬────────────┘
                                 │ *graph.Graph
                                 ▼
                        ┌─────────────────────┐
                        │ semantic.Pipeline    │
                        │ [Token, Proxy,       │
                        │  Oracle, Governance, │
                        │  Treasury].Run(g)    │
                        └────────┬────────────┘
                                 │ annotated *graph.Graph
                                 ▼
                        ┌─────────────────────┐
                        │ cyclonedx.Builder    │
                        │ Build(g) → *cdx.BOM  │
                        └────────┬────────────┘
                                 │ CycloneDX 1.6 JSON
                                 ▼
                           output/bom.json
```

### Data Flow: Information Movement

| Stage | Input | Output | Package |
|---|---|---|---|
| Repository | URL/local path | `git.Repository{Path}` | `internal/git` |
| Discovery | `repo.Path` | `Project{SolidityFiles[],...}` | `internal/discovery` |
| Parse | `.sol` file path | `*ParsedFile{Contracts[]{Name,Kind,Imports,Inherits,Functions,Events,Modifiers}}` | `internal/parser/solidity` |
| Graph Build | `[]*ParsedFile` | `*graph.Graph{Nodes map[id→Node{Metadata}], Edges[]{From,To,Relationship}}` | `internal/graph` |
| Semantic Analysis | `*graph.Graph` | Mutated `*graph.Graph` with `node.Metadata["ComponentType"]`, `TokenStandard`, `ProxyPattern`, `OracleProvider` | `internal/semantic` |
| BOM Export | `*graph.Graph` | `*cdx.BOM` serialised to JSON | `internal/cyclonedx` |

---

## PHASE 2: Current Capability Inventory

### Capability 1: Git Repository Acquisition
- **Purpose:** Shallow-clone a remote GitHub repository or accept a local path
- **Input:** URL string or local filesystem path, optional branch/tag ref
- **Output:** `git.Repository{URL, Path, Branch}`
- **Location:** `internal/git/manager.go:45` (`CloneWithRef`)
- **Maturity:** **Functional** — shallow clone (depth=1), re-uses existing clone if present, cleanup via `os.RemoveAll`. No authentication beyond public HTTPS. No tag support (only branch references via `plumbing.NewBranchReferenceName`).

### Capability 2: Filesystem Discovery
- **Purpose:** Walk a repository and classify files by blockchain type
- **Input:** Root directory path
- **Output:** `discovery.Project{SolidityFiles, VyperFiles, RustFiles, MoveFiles, ConfigFiles}`
- **Location:** `internal/discovery/scanner.go:38` (`Scan`)
- **Maturity:** **Functional** — correctly skips `node_modules`, `.git`, `artifacts`, `cache`, `out`, `lib`, `dist`, `build`, `.foundry`. Detects `.sol`, `.vy`, `Cargo.toml`, `Move.toml`, `foundry.toml`, `hardhat.config.*`, `package.json`. No depth limit by default.

### Capability 3: Solidity Parsing (Regex-based)
- **Purpose:** Extract structural declarations from `.sol` files without a compiler
- **Input:** Path to `.sol` file
- **Output:** `*parser.ParsedFile{Path, Language, FileImports, Contracts[{Name, Kind, Imports, Inherits, Functions, Events, Modifiers}]}`
- **Location:** `internal/parser/solidity/parser.go:57` (`Parse`)
- **Maturity:** **Partially functional** — handles multi-line declarations, comment stripping (`//` and `/* */`), all import forms, constructor-argument inheritance (`ERC20('Token','SYM'), Ownable`), function visibility/mutability. **Cannot parse:** function bodies, state variables, storage layout, types, expressions, mappings, structs, ABI, `using for` directives, assembly blocks.

### Capability 4: Dependency Graph Construction
- **Purpose:** Build a directed graph of contract relationships
- **Input:** `[]*parser.ParsedFile`
- **Output:** `*graph.Graph` with nodes (contracts) and edges (inherits, imports)
- **Location:** `internal/graph/builder.go:21` (`Build`)
- **Maturity:** **Functional** — two-pass (register all contracts first, then create edges). Creates external stub nodes for unresolved imports. Package name inference for OpenZeppelin, Chainlink, Uniswap, Aave. Graph supports: `DependenciesOf`, `DependentsOf`, `TopologicalSort`, `NodesByType`, `EdgesFrom`, `Stats`.

### Capability 5: Semantic Contract Classification
- **Purpose:** Classify each contract node by its DeFi protocol role
- **Input:** `*graph.Graph`
- **Output:** Annotated graph (mutations to `node.Metadata`)
- **Location:** `internal/semantic/`
- **Maturity:** **Partially implemented (pattern matching only)**

| Analyzer | Detection Signals | Metadata Produced | Location |
|---|---|---|---|
| TokenAnalyzer | Inheritance: `ERC20`, `ERC721`, `ERC1155`; Name: contains `TOKEN` or `NFT` | `ComponentType=Token`, `TokenStandard=ERC20\|ERC721\|ERC1155` | `token.go` |
| ProxyAnalyzer | Inheritance: UUPS/Transparent/Beacon/ERC1967 bases; Name: contains `proxy` or `upgradeable` | `ComponentType=Proxy`, `Upgradeable=true`, `ProxyPattern=UUPS\|Transparent\|Beacon\|Unknown` | `proxy.go` |
| OracleAnalyzer | Inheritance: AggregatorV3Interface etc.; Functions: `latestrounddata`, `getprice`, etc.; Imports: `chainlink`, `aggregator` | `ComponentType=Oracle`, `OracleProvider=Chainlink\|Uniswap TWAP\|Band Protocol\|Unknown` | `oracle.go` |
| GovernanceAnalyzer | Inheritance: Governor/TimelockController/AccessControl; Name: contains `governor`, `dao`, `timelock` | `ComponentType=Governance` | `governance.go` |
| TreasuryAnalyzer | Inheritance: Gnosis/SafeVault; Name: contains `treasury`, `vault`, `safe`, `multisig`, `escrow`, `custody`, `reserve` | `ComponentType=Treasury` | `treasury.go` |

### Capability 6: CycloneDX 1.6 BOM Export
- **Purpose:** Serialize the annotated graph as a standards-compliant BOM
- **Input:** `*graph.Graph`
- **Output:** `*cdx.BOM` written to JSON file
- **Location:** `internal/cyclonedx/builder.go:30` (`Build`)
- **Maturity:** **Functional** — uses official `cyclonedx-go` v0.9.0. UUID serial number, RFC3339 timestamp, tool metadata (`SmartBOM v0.1.0`). Custom properties under `smartbom:` namespace: `ContractType`, `TokenStandard`, `Upgradeable`, `ProxyPattern`, `OracleProvider`, `PackageName`, `SourceFile`, `Inherits`. Dependency mapping from graph edges.

### Capability 7: Vulnerability Scanner Framework (stubs only)
- **Purpose:** Extensible scanning engine for future security analysis
- **Input:** `*graph.Graph` (per-scanner `Scan` method)
- **Output:** `[]Finding{ID, Severity, Component, Title, Description, Remediation, References}`
- **Location:** `internal/vuln/scanner.go`
- **Maturity:** **Framework only — zero findings produced** — 5 stub scanners: `DependencyScanner` (OSV/Snyk placeholder), `ReentrancyScanner`, `DelegatecallScanner`, `AccessControlScanner`, `UpgradeabilityScanner`. All `Scan()` methods return `nil, nil`.

### Capabilities Explicitly NOT Implemented
- Vyper parsing (stubs only — files are discovered but skipped)
- Rust/Move parsing (files discovered, no parsers)
- `graph` command (returns error)
- `vuln` command (returns error)
- Any CBOM cryptographic detection
- Any PQC analysis
- Version tracking for contracts
- NPM/Cargo/Move dependency version resolution
- On-chain address mapping or RPC integration
- Hash integrity of source files

---

## PHASE 3: Domain Model Analysis

### Entity Inventory

**Critical structural observation:** There are two parallel type hierarchies. The `internal/model` package defines `Component` and `Dependency` types, but these are **not used** in the runtime pipeline. The actual data types in the pipeline are `graph.Node` and `graph.Edge`.

| Entity | Defined In | Used In Pipeline? | Notes |
|---|---|---|---|
| `model.Component` | `internal/model/component.go:45` | **No** — orphaned | Constants (`ComponentType`, `TokenStandard` strings) are used by semantic analyzers, but the struct itself is not. |
| `model.Dependency` | `internal/model/dependency.go:14` | **No** — orphaned | `RelationshipKind` constants (`RelImports`, `RelInherits`, `RelUses`, `RelDeploys`) are defined but not referenced anywhere in the codebase. |
| `graph.Node` | `internal/graph/graph.go:11` | **Yes** — primary runtime entity | `ID` = contract name, `Type` = kind string, `Metadata` = open `map[string]any` |
| `graph.Edge` | `internal/graph/graph.go:18` | **Yes** — primary runtime entity | `From`, `To`, `Relationship` = "imports"\|"inherits" |
| `parser.ParsedFile` | `internal/parser/interface.go:6` | **Yes** — parse output | `Path`, `Language`, `FileImports`, `Contracts[]` |
| `parser.Contract` | `internal/parser/interface.go:13` | **Yes** — per-contract parse result | `Name`, `Kind`, `Imports`, `Inherits`, `Functions`, `Events`, `Modifiers`, `SourceFile` |
| `parser.Function` | `internal/parser/interface.go:26` | **Yes** — function signature | `Name`, `Visibility`, `Mutability` |
| `discovery.Project` | `internal/discovery/scanner.go:12` | **Yes** — discovery output | File lists by language |
| `git.Repository` | `internal/git/manager.go:15` | **Yes** — clone result | `URL`, `Path`, `Branch` |
| `vuln.Finding` | `internal/vuln/scanner.go:24` | **No** — never produced | `ID`, `Severity`, `Component`, `Title`, `Description`, `Remediation`, `References` |

### Entity Relationship Map

```
URL/Path
  └─[clones]──▶ git.Repository
                     └─[discovered by]──▶ discovery.Project
                                               └─[parsed to]──▶ parser.ParsedFile[]
                                                                       └─[contains]──▶ parser.Contract[]
                                                                                              └─[becomes]──▶ graph.Node
                                                                                              └─[produces]──▶ graph.Edge (inherits, imports)

graph.Graph
  ├─ Nodes (map[string]*Node)
  │     └─[annotated by]──▶ semantic.Analyzer (Token/Proxy/Oracle/Governance/Treasury)
  └─ Edges ([]{From, To, Relationship})

graph.Graph
  └─[exported by]──▶ cyclonedx.Builder ──▶ *cdx.BOM ──▶ output/bom.json
```

### Key Relationship Observations

1. Only `"inherits"` and `"imports"` edge types are produced, despite `"uses"` and `"deploys"` being defined in `model.Dependency`.
2. Functions, events, and modifiers are stored as `[]string` in `node.Metadata` — they are **not** first-class graph nodes. This is the primary architectural limitation.
3. The graph node ID equals the contract name (e.g., `"MyToken"`). No namespace or source-file disambiguation. Two contracts with the same name from different files will collide — the second `UpsertNode` overwrites the first.

---

## PHASE 4: Blockchain Intelligence Assessment

### Smart Contracts
**Classification: Partially Implemented**

- **Evidence:** `internal/parser/solidity/parser.go` extracts `Name`, `Kind`, `Imports`, `Inherits`, `Functions`, `Events`, `Modifiers`. `internal/graph/builder.go` creates nodes and edges.
- **Gap:** No function body analysis. No state variables. No ABI types. No expression parsing. No visibility/mutability enforcement.

### Protocol Architecture
**Classification: Partially Implemented**

- **Evidence:** Five semantic analyzers classify contracts into DeFi roles. `OracleAnalyzer` distinguishes Chainlink vs TWAP vs Band. `GovernanceAnalyzer` recognises multi-inheritance Compound-style Governor.
- **Gap:** Classification is by name/inheritance string matching, not by understanding of inter-contract interactions. No concept of "Protocol" as an aggregate entity spanning multiple contracts.

### Upgradeability Patterns
**Classification: Partially Implemented**

- **Evidence:** `internal/semantic/proxy.go` — `proxyBasePatterns` includes `transparentupgradeableproxy`, `uupsupgradeable`, `beaconproxy`, `upgradeablebeacon`, `erc1967proxy`, `proxyadmin`. Sets `ProxyPattern` metadata.
- **Gap:** Detection is purely by inheritance label matching. No storage layout analysis. No initializer guard detection. No implementation address tracking. `detectProxyPattern` does not return a case for `erc1967proxy` specifically — it falls through to `"Unknown"`.

### Proxy Patterns
**Classification: Partially Implemented** (same as upgradeability — they are the same analyzer)

### Oracle Integrations
**Classification: Partially Implemented**

- **Evidence:** `internal/semantic/oracle.go` — three detection signals: inheritance (`AggregatorV3Interface`, `FeedRegistryInterface`, `IPriceFeed`), function names (`latestrounddata`, `getprice`, `latestanswer`), import path substrings (`chainlink`, `aggregator`, `pricefeed`, `oracle`). Distinguishes Chainlink, Band, Uniswap TWAP.
- **Gap:** No understanding of oracle data freshness, staleness thresholds, aggregation methods, or round data semantics.

### Governance Systems
**Classification: Partially Implemented**

- **Evidence:** `internal/semantic/governance.go` — detects `Governor`, `GovernorBravo`, `GovernorVotes`, `TimelockController`, `AccessControl`, `AccessControlEnumerable`. Also catches names containing `governor`, `governance`, `dao`, `timelock`.
- **Gap:** No extraction of voting parameters, quorum fractions, proposal lifecycle states, or token-weighting mechanisms.

### Treasury Systems
**Classification: Partially Implemented**

- **Evidence:** `internal/semantic/treasury.go` — name patterns: `treasury`, `vault`, `safe`, `multisig`, `escrow`, `custody`, `reserve`. Base patterns: `gnosis`, `multisigwallet`, `safevault`.
- **Gap:** No multi-sig threshold extraction, no asset-type inventory, no ETH vs ERC-20 custody distinction.

### Access Control Systems
**Classification: Not Implemented as a separate concern**

- **Evidence:** `GovernanceAnalyzer` detects `AccessControl` contracts and classifies them as `Governance`. No separate access control analyzer exists.
- **Gap:** No extraction of which roles exist, which functions are role-protected, what the role hierarchy is. The vulnerability scanner stub `AccessControlScanner` exists but produces zero findings.

### Summary Table

| Category | Status | Key Evidence |
|---|---|---|
| Smart contract discovery | Implemented | `discovery/scanner.go`, `parser/solidity/parser.go` |
| Protocol classification | Partially Implemented | `semantic/` — 5 analyzers, pattern-matching only |
| Upgradeability patterns | Partially Implemented | `semantic/proxy.go` |
| Proxy patterns | Partially Implemented | `semantic/proxy.go` |
| Oracle integrations | Partially Implemented | `semantic/oracle.go` |
| Governance systems | Partially Implemented | `semantic/governance.go` |
| Treasury systems | Partially Implemented | `semantic/treasury.go` |
| Access control systems | Not Implemented | No dedicated analyzer; no role/function mapping |
| Inter-contract call graphs | Not Implemented | No function body parsing |
| Storage layout analysis | Not Implemented | No AST parser |
| ABI extraction | Not Implemented | No AST parser |

---

## PHASE 5: Cryptography Analysis Audit

**Finding: Zero cryptographic detection capability exists.**

A full search of all source files reveals:

- No regular expressions matching `ecdsa`, `keccak`, `sha256`, `sha3`, `ecrecover`, `ed25519`, `bls`, `signature`, `verify`, `merkle`, `eip712`, `erc1271`, `encrypt`, `decrypt`.
- No import patterns for cryptographic libraries (`ECDSA.sol`, `MerkleProof.sol`, `EIP712.sol`, `SignatureChecker.sol`).
- No function-name patterns for cryptographic operations.
- No algorithm classification tables.
- No key-size or curve-name detection.

### What the Output BOM Shows (and What It Does Not Mean)

The actual `output/bom.json` (produced by scanning the OpenZeppelin monorepo) includes a component named `SignerRSA` with `bom-ref: SignerRSA` and `SourceFile: .../cryptography/signers/SignerRSA.sol`. **This is coincidental** — the component appears because the Solidity parser found a contract named `SignerRSA` in a `.sol` file. The system has no knowledge that this is a cryptographic contract. The component carries no cryptographic metadata properties. It has only `smartbom:SourceFile` and `smartbom:Inherits: AbstractSigner`.

### Specific Primitive Assessment

| Primitive | Detected? | How | Evidence |
|---|---|---|---|
| ECDSA | No | — | No patterns anywhere |
| RSA | No | — | `SignerRSA` appears in BOM by coincidence of contract name; no RSA-specific detection |
| Ed25519 | No | — | — |
| BLS | No | — | — |
| Keccak256 | No | — | — |
| SHA-256 | No | — | — |
| Merkle Proofs | No | — | — |
| Signature Verification | No | — | — |
| EIP-712 | No | — | — |
| ERC-1271 | No | — | — |

### CBOM Existence

**No CBOM functionality exists.** The `smartbom:schema = "blockchain-cbom"` property in `internal/cyclonedx/builder.go:72` is a label in the metadata header — not an implementation. It declares intent without providing substance.

---

## PHASE 6: SBOM Maturity Assessment

### Score: 28 / 100

| SBOM Principle | Status | Score Weight | Evidence |
|---|---|---|---|
| Component Inventory | Present — all Solidity contracts become CycloneDX components | 15/20 | `cyclonedx/builder.go:76` |
| Dependency Inventory | Present — import/inheritance edges become CycloneDX dependencies | 12/20 | `cyclonedx/builder.go:101` |
| Version Tracking | Absent — `Version` field in `graph.Node.Metadata` is never set | 0/15 | Pragma `^0.8.20` parsed but not extracted or stored |
| External Dependency Provenance | Absent — external nodes get `PackageName` only, no PURL, no version, no hash | 1/15 | `graph/builder.go:88` — `PackageName` only |
| License Information | Absent — SPDX identifiers in source files not extracted | 0/10 | Every `.sol` test file has `// SPDX-License-Identifier: MIT`; ignored |
| Source Integrity (Hashes) | Absent — no file hashing | 0/10 | — |
| Relationship Modeling | Partial — `imports` and `inherits` only; `uses` and `deploys` absent | 5/10 | `model/dependency.go:6` defines but never creates `uses`/`deploys` |
| CycloneDX Spec Compliance | Partial — SpecVersion 1.6 correctly used; many standard fields empty (purl, licenses, hashes, externalReferences, supplier) | 8/15 | `cyclonedx/builder.go` |
| Transitive Dependencies | Absent — only direct imports resolved; no recursive following | 0/10 | `graph/builder.go:75` — one level only |

**Justification for 28/100:** The core mechanics work end-to-end — a component is created for every discovered contract, dependencies are emitted, and the file is spec-compliant JSON. However, every field that gives SBOM its actual supply-chain value — versions, PURLs, licenses, hashes, suppliers, transitive dependencies — is absent. The output is structurally valid but informationally sparse.

---

## PHASE 7: CBOM Maturity Assessment

### Score: 2 / 100

| CBOM Principle | Status | Evidence |
|---|---|---|
| Cryptographic inventory | Not implemented | Zero detection capability |
| Algorithm identification | Not implemented | No algorithm patterns, no detection logic |
| Algorithm usage tracking | Not implemented | No function-level or call-site analysis |
| Security property tracking | Not implemented | No key size, curve name, hash output length |
| Key management visibility | Not implemented | — |
| Trust assumption documentation | Not implemented | — |
| EIP-712 typed data signing | Not implemented | — |
| ERC-1271 signature verification | Not implemented | — |
| Quantum vulnerability classification | Not implemented | — |

**Justification for 2/100:** The 2 points reflect that (a) the `smartbom:schema = "blockchain-cbom"` property declares intent in the output schema, and (b) the underlying infrastructure — the `graph.Node.Metadata` property bag, the `semantic.Analyzer` pipeline interface, and the CycloneDX property serialization in `buildProperties()` — is structurally capable of receiving CBOM data. The platform is ready to be extended; the extension has not been written.

---

## PHASE 8: PQC Readiness Assessment

### Current PQC Support: 0%

No NIST PQC mappings, no CNSA 2.0 alignment, no quantum risk scoring, no migration guidance, no post-quantum algorithm detection.

### Existing Reusable Components for Future PQC Work

| Component | Location | Reuse Path |
|---|---|---|
| `graph.Node.Metadata` open map | `internal/graph/graph.go:13` | Add `PQCRisk`, `QuantumVulnerability`, `NISTRecommendation` metadata keys without schema changes |
| `semantic.Analyzer` interface | `internal/semantic/interface.go:7` | A `CryptoAnalyzer` and `PQCAnalyzer` plug into `DefaultPipeline()` |
| `semantic.Pipeline.Run()` | `internal/semantic/interface.go:26` | Sequential analyzer execution already supports ordering (crypto must run before PQC) |
| `buildProperties()` | `internal/cyclonedx/builder.go:122` | Add `smartbom:NISTAlgorithm`, `smartbom:QuantumRisk`, `smartbom:CNSAStatus` property emission |

### Missing Components

1. **Cryptographic primitive detector** — The entire CBOM layer (prerequisite for PQC)
2. **Algorithm-to-NIST mapping table** — e.g., `secp256k1 ECDSA → id-ecPublicKey (OID 1.2.840.10045.2.1)`, quantum status: vulnerable
3. **Quantum vulnerability classifier** — ECDSA/RSA/DH → quantum-vulnerable; SHA-256/Keccak-256 → weakened but not broken; AES-128 → weakened
4. **CNSA 2.0 timeline data** — per-algorithm migration deadlines
5. **PQC migration guidance generator** — For each vulnerable primitive, recommend replacement (e.g., CRYSTALS-Kyber for key exchange, CRYSTALS-Dilithium for signatures)

### Required Architectural Changes

1. Add `CryptoAnalyzer` to the semantic pipeline
2. Add `PQCAnalyzer` (depends on `CryptoAnalyzer` output; must run after it)
3. Promote function-level cryptographic usage to first-class graph relationships
4. Add CycloneDX CBOM-extension properties to the BOM output

### Effort Estimate

- **CBOM Phase 1** (import/inheritance-based crypto detection): 2–4 weeks
- **CBOM Phase 2** (function-body crypto detection — requires AST parser): 2–3 months
- **PQC overlay on CBOM**: 1–2 months after CBOM Phase 2
- **Total to meaningful PQC intelligence**: ~4–6 months

---

## PHASE 9: Knowledge Graph Readiness

### Current Graph Characteristics

The `internal/graph` package implements a conventional directed graph — not a knowledge graph. It has:

| Feature | Present? | Location |
|---|---|---|
| Directed edges with typed relationships | Yes | `graph.go:18` — `Relationship` field |
| Open-schema node metadata | Yes | `graph.go:13` — `map[string]any` |
| Graph traversal (DFS/BFS-equivalent) | Yes | `graph.go:78` — `DependenciesOf`, `DependentsOf` |
| Topological ordering | Yes | `graph.go:110` — Kahn's algorithm |
| Node type query | Yes | `graph.go:148` — `NodesByType` |
| Cycle detection | Yes | `graph.go:141` — implicit in topological sort |
| First-class `Function` nodes | No | Functions stored as `[]string` in metadata |
| First-class `Event` nodes | No | Events stored as `[]string` in metadata |
| First-class `CryptographicPrimitive` nodes | No | Not implemented at all |
| `Repository → File → Contract` provenance chain | No | `SourceFile` stored as metadata string; no `File` node type |
| `calls` / `emits` / `uses` edge relationships | No | Only `imports` and `inherits` edges are created |
| Multi-hop path queries | No | No path-finding algorithm; only single-hop traversal |

### Target Knowledge Graph Model

```
Repository
  └─[contains]──▶ File
                     └─[defines]──▶ Contract
                                        ├─[inherits]──▶ Contract/ExternalBase
                                        ├─[imports]───▶ Package
                                        └─[exposes]───▶ Function
                                                              ├─[calls]────▶ Function
                                                              └─[uses]─────▶ CryptographicPrimitive
                                                                                    ├─[maps_to]──▶ NISTAlgorithm
                                                                                    └─[classified]▶ QuantumVulnerable
```

### Gap to Knowledge Graph

What exists:
- `Contract → [inherits] → ExternalBase`
- `Contract → [imports] → Package`
- Semantic labels on Contract nodes

What is missing:
- `Repository`, `File`, `Protocol` as first-class nodes
- `Function` as first-class nodes with `calls` edges
- `CryptographicPrimitive` nodes
- `NISTAlgorithm`, `QuantumRiskLevel` nodes
- Path-finding across the graph (shortest path, reachability)
- Reverse-index queries ("which contracts use ECDSA?")

---

## PHASE 10: Gap Analysis

| Dimension | Current State | Target State | Gap | Complexity | Priority |
|---|---|---|---|---|---|
| **Traditional SBOM** | Components + dependency structure; no versions, no PURLs, no licenses, no hashes, no transitive deps | Full CycloneDX SBOM: versions from pragma/package.json, PURL for npm/foundry packages, SPDX licenses, SHA-256 hashes, recursive import resolution | Medium — data is present in source files and package manifests; requires extraction logic | Medium | P1 — High |
| **Blockchain SBOM** | 5 pattern-matching semantic classifiers; no ABI, no deployed addresses, no protocol grouping | Contract-level BOM with ABI, deployment addresses, protocol topology, upgrade graph, cross-chain metadata | ABI requires AST/solc; addresses require on-chain RPC; protocol grouping requires a new abstraction layer | High | P1 — High |
| **Protocol Intelligence** | Name/inheritance pattern matching for 5 categories | Full protocol topology: which contracts call which, admin role graphs, upgrade paths, governance parameter extraction | Requires function body parsing (AST), call graph construction, role graph extraction | Very High | P2 — Medium |
| **CBOM** | Zero implementation (schema label only) | Cryptographic inventory per-contract: algorithms detected, usage context (which function uses which primitive), key sizes, EIP-712/ERC-1271, trust assumptions | Import/inheritance detection is medium effort; function-level detection requires AST | High | P0 — Critical |
| **PQC Intelligence** | Zero implementation | NIST/CNSA 2.0 algorithm mapping, quantum vulnerability scoring per contract and per protocol, migration priority ordering | Depends entirely on CBOM implementation as prerequisite | Very High | P2 — Medium (unblocks after CBOM) |

---

## PHASE 11: Strategic Recommendations

### Quick Wins (1–2 weeks)

**QW-1: SPDX License Extraction**
The Solidity parser (`internal/parser/solidity/parser.go`) reads every line. Every contract in the test suite has `// SPDX-License-Identifier: MIT`. Add one regex, store the result in `node.Metadata["License"]`, emit it as a CycloneDX `licenses` entry. Zero new dependencies. Directly fills the largest single SBOM metadata gap.

**QW-2: Pragma / Solidity Version Extraction**
`pragma solidity ^0.8.20` is present in every file and is parsed-but-ignored. Extracting it populates `node.Metadata["Version"]` and the CycloneDX `Version` field. The parser infrastructure already exists.

**QW-3: Source File SHA-256 Integrity Hash**
During parsing, compute `sha256.Sum256(fileBytes)` and store it as `node.Metadata["SourceHash"]`. Emit as a CycloneDX component `hashes` entry. Provides supply-chain integrity anchoring with zero new dependencies (Go stdlib).

**QW-4: NPM Package Version Resolution from package.json**
`package.json` is already discovered by `FileScanner`. Parse it to extract `dependencies["@openzeppelin/contracts"]` etc., then populate the `Version` field on external stub nodes. This directly solves the SBOM provenance gap for NPM-based Solidity projects.

**QW-5: Fix model.Component Orphaning**
`internal/model/component.go` and `internal/model/dependency.go` define types that are never instantiated in the pipeline. Either (a) replace `graph.Node`/`graph.Edge` with `model.Component`/`model.Dependency` throughout, or (b) delete the model package and consolidate constants into the semantic package. The split creates confusion about the canonical data model.

**QW-6: DOT Graph Export**
The `graph` command (`cmd/smartbom/cmd/graph_cmd.go`) returns an error. The `graph.Graph` data structure is already complete. Generating a Graphviz DOT file is ~30 lines. This enables immediate visualization of discovered dependency relationships.

---

### Medium Effort (1–2 months)

**ME-1: CBOM Phase 1 — Import/Inheritance-based Cryptographic Primitive Detection**
Add a `CryptoAnalyzer` to `semantic/`. Use the same pattern-matching approach as existing analyzers but targeting cryptographic signals:

- Import patterns: `ECDSA.sol`, `MerkleProof.sol`, `EIP712.sol`, `SignatureChecker.sol`, `RSA.sol`
- Inheritance patterns: `EIP712`, `AbstractSigner`, `ERC1271`
- Function-name patterns: `ecrecover`, `keccak256`, `sha256`, `verify`, `hashTypedDataV4`

Output: `node.Metadata["CryptoPrimitives"] = []string{"ECDSA", "Keccak256", "EIP712"}`. This is the architectural foundation for all CBOM and PQC work.

**ME-2: Vyper Import/Name Parsing**
At minimum, extract `# @version`, `from X import Y`, and contract-equivalent `@external def functionName()` from `.vy` files. The `Parser` interface already exists. Vyper is heavily used in DeFi (Curve, Yearn); leaving it a stub significantly limits coverage.

**ME-3: Access Control Inventory Analyzer**
The parser already extracts `Modifiers` and `Functions`. Add a `SecurityAnalyzer` that correlates them: which functions have `onlyOwner`, `onlyRole(ADMIN_ROLE)`, `whenNotPaused`, etc. Output structured access control metadata per function. This transforms the tool from "what kind of contract is this?" to "who can do what in this protocol?"

**ME-4: Foundry/Hardhat Config Parsing**
`foundry.toml` and `hardhat.config.*` are already discovered. Parsing them yields: installed library versions (for Foundry `lib/`), compiler version, optimizer settings, network configurations. This data directly populates SBOM version fields and provides build provenance.

**ME-5: Transitive Dependency Resolution**
Currently `graph/builder.go:75` processes only direct imports. For cloned repositories where `lib/` (Foundry) or `node_modules/` (Hardhat) is available, follow imports recursively. The clone already makes the files available on disk. This closes the transitive dependency gap — critical for SBOM completeness.

**ME-6: Vulnerability Scanner Activation (OSV Integration)**
The `DependencyScanner` stub in `internal/vuln/scanner.go:71` has a `// TODO: integrate with OSV / Snyk` comment. The OSV API accepts package name + ecosystem + version and returns CVEs. With versions populated (ME-4), activating this scanner requires only an HTTP client and JSON parsing.

---

### Major Initiatives (3–6 months)

**MI-1: AST-based Solidity Parser (Architectural Prerequisite)**
The single highest-leverage architectural change. Replace the regex parser with a proper Solidity grammar parser. Options:
- **tree-sitter-solidity** via CGo bindings (battle-tested, used by GitHub for code navigation)
- **ANTLR4 Go target** with the Solidity grammar from `soliditylab/antlr4-solidity`
- **solc JSON input/output** via subprocess (most accurate but adds a compiler dependency)

This unlocks: state variable extraction, function body analysis, internal/external call graph construction, type information, storage layout (critical for proxy safety analysis), ABI generation, and expression-level cryptographic primitive detection.

**MI-2: CBOM Phase 2 — Function-level Cryptographic Analysis**
With AST parsing available, detect cryptographic usage at the call-site level:
- `keccak256(abi.encode(...))` in function bodies
- `ecrecover(hash, v, r, s)` calls
- `ECDSA.recover(hash, signature)` calls
- `_hashTypedDataV4(structHash)` calls
- Storage of keys/secrets

Each detected usage becomes a `CryptographicPrimitive` node in the graph with edges to the function that uses it. This is the true CBOM — not import detection but usage tracking.

**MI-3: NIST/CNSA 2.0 PQC Intelligence Layer**
Build on MI-2:
1. **Algorithm registry table**: map detected primitives to NIST OIDs and CNSA 2.0 categories
2. **Quantum risk classifier**: per-algorithm quantum vulnerability and timeline
3. **Risk propagation**: if Contract A uses ECDSA and Contract B inherits A, B also has ECDSA exposure
4. **PQC migration report**: ranked by risk, with recommended NIST PQC replacements (CRYSTALS-Dilithium for signatures, CRYSTALS-Kyber for key exchange)
5. **CycloneDX output**: emit structured `smartbom:pqc:*` properties per component

**MI-4: Knowledge Graph Upgrade**
Promote functions, events, cryptographic primitives, and protocols to first-class graph nodes:

```
New node types: File, Protocol, Function, Event, CryptoPrimitive, NISTAlgorithm
New edge types: defines, exposes, calls, emits, uses, maps_to
```

This enables queries such as:
- "Show all contracts that directly or transitively use ECDSA"
- "What is the call path from governance.execute() to ECDSA.recover()?"
- "Which protocols have quantum-vulnerable cryptographic dependencies?"

**MI-5: On-Chain Protocol Intelligence**
For deployed protocols: integrate Etherscan/Blockscout APIs to map contract names to addresses, verify source code, resolve ERC-1967 proxy `_IMPLEMENTATION_SLOT` to detect live implementation contracts, and cross-reference with known protocol deployments. This transforms the tool from a source-analysis platform into a live protocol intelligence system.

---

## Summary Assessment

| Platform Dimension | Current Maturity | Gap to Vision |
|---|---|---|
| Traditional SBOM | 28/100 — structure present, metadata absent | Medium effort: version/license/PURL extraction |
| Blockchain SBOM | 35/100 — semantic classifiers functional; ABI/addresses absent | High effort: requires AST parser |
| Protocol Intelligence | 15/100 — pattern matching only, no call graph | Very high effort: requires AST + call graph |
| CBOM | 2/100 — intent declared, zero implementation | Critical path: Phase 1 feasible in weeks |
| PQC Intelligence | 0/100 — entirely absent | Depends on CBOM; 4–6 months total |

**The architectural foundations — the pipeline, the extensible graph, the open metadata system, the analyzer interface, the CycloneDX property mechanism — are sound and well-tested.** The codebase is clean Go, idiomatically structured, with meaningful test coverage on the implemented components.

**The most important next action is CBOM Phase 1.** The existing `semantic.Analyzer` interface means a `CryptoAnalyzer` can be built using exactly the same pattern as the five existing analyzers, requiring no architectural changes. Without it, the tool is an interesting blockchain SBOM prototype. With it, the path to PQC intelligence becomes a roadmap rather than a vision.
