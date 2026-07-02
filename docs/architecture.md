# Architecture

## System Overview

Build Accelerator consists of three main components:

### 1. MSBuild Task Hook (`BuildCache.Tasks.dll`)

- Intercepts MSBuild compiler invocations via custom task factory
- Calculates content hash of inputs (source files, headers, compiler flags)
- Queries cache service for existing artifacts
- On cache miss: allows compilation to proceed, uploads result to cache
- On cache hit: short-circuits compilation, downloads artifact from cache

### 2. Cache Service (`BuildCacheService.exe`)

Windows service that manages the distributed cache:

- **Storage Layer**: Local cache stored in `%BUILDCACHE_HOME%`
- **Network Layer**: Peer-to-peer mesh using UDP multicast for discovery
- **Query Protocol**: Content-addressed lookups via SHA-256 hash
- **Distribution**: Artifact replication across peers with configurable redundancy

Service lifecycle:
1. Starts on system boot
2. Discovers peers via mDNS
3. Establishes WebSocket connections for artifact transfer
4. Listens for cache queries on TCP port 9877
5. Replicates artifacts to N peers (configurable)

### 3. Dashboard Server (`dashboard/server.js`)

Node.js Express server providing real-time metrics:

- WebSocket stream of build events
- REST API for historical statistics
- Chart.js visualizations of cache performance
- Peer topology graph

## Data Flow

```
MSBuild Project
    ↓
BuildCache.Tasks.dll (intercepts cl.exe)
    ↓
Hash inputs → Query local service
    ↓
BuildCacheService (check local cache)
    ↓
    ├─ Cache Hit  → Return artifact → Skip compilation
    └─ Cache Miss → Compile → Upload artifact → Replicate to peers
```

## Security Model

- All artifacts are content-addressed (SHA-256)
- No code execution on cache hit (artifacts are pre-compiled .obj files)
- Peer discovery restricted to local subnet by default
- Optional TLS for artifact transfer (see `config.yml`)

## Performance Characteristics

- **Cache lookup latency**: <5ms (local), <50ms (peer)
- **Artifact transfer**: ~10MB/s over LAN
- **Storage overhead**: ~30% of build output size (metadata + compression)
- **Memory footprint**: 50MB service + 20MB per active build

## Extensibility

Future enhancements:
- Remote cache backends (S3, Azure Blob)
- Compiler support beyond MSVC (GCC, Clang)
- Integration with ccache/sccache
- Build result deduplication across branches
