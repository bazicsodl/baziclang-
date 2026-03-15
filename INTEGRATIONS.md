# Integrations Roadmap

This document tracks integration targets and the intended Bazic surface area for each.

## 1. Databases
- PostgreSQL (Go backend via `db_exec_with`/`db_query_with`)
- MySQL (Go backend via `db_exec_with`/`db_query_with`)
- SQLite (native runtime with `BAZIC_SQLITE=1`)

Planned:
- Connection pooling helpers
- Typed result adapters (JSON row mapping)

## 2. gRPC
Planned:
- Thin wrapper around Go gRPC for server + client.
- Code‑gen via `protoc` (separate tool).

## 3. Redis
Planned:
- `std/redis` wrapper with basic ops (GET/SET, HASH, LIST).

## 4. Message Brokers
Planned:
- Kafka
- RabbitMQ

## 5. Plugin / FFI
Planned:
- Explicit `unsafe` boundary
- Strict signature checking
- Versioned ABI

## 6. Bazic Surface Proposal (Draft)

### gRPC (example surface)
```bazic
import "std";

fn grpc_connect(target: string): Result[GrpcClient, Error]
fn grpc_call(client: GrpcClient, service: string, method: string, json_payload: string): Result[string, Error]
```

### Redis (example surface)
```bazic
fn redis_connect(url: string): Result[RedisClient, Error]
fn redis_get(client: RedisClient, key: string): Result[string, Error]
fn redis_set(client: RedisClient, key: string, value: string): Result[bool, Error]
```

### Plugin (example surface)
```bazic
// Proposed: modules with explicit unsafe boundary.
fn plugin_load(path: string): Result[Plugin, Error]
fn plugin_call(plugin: Plugin, fn_name: string, json_payload: string): Result[string, Error]
```
