# SAI

SAI is a backend intent compiler. The current implementation delivers the first production slice of the pipeline:

`.sai -> parser -> AST -> IR -> planner`

This repository is implemented in Go so the CLI ships as a small native binary and the core compiler stays fast, deterministic, and easy to distribute.

## Project layout

```text
ast/
cli/
compiler/
  build/
  deploy/
  incident/
  infra/
  runtime/
executor/
examples/
integrations/
  bitbucket/
  github/
ir/
parser/
planner/
utils/
main.go
```

## MVP status

Implemented now:

- Project scaffold with strict module boundaries
- CLI commands: `init`, `validate`, `plan`, `deploy`, `logs`, `rollback`
- `.sai` lexer and parser
- AST types for app, service, and resource declarations
- AST to typed IR normalization with defaults and validation
- Deterministic planner that maps `users + budget` to a deployment profile

Deferred to later slices:

- Terraform JSON generation
- Dockerfile generation
- Runtime config lowering
- Deployment execution
- Incident execution

## Supported manifest shape

```sai
app "orders" {
  users 5000
  budget 75usd
  env prod
}

service api {
  runtime node
  path "server"
  port 3000
  public http
  connects postgres
}

database postgres {
  type managed
  size small
}
```

## Normalization rules

- `app.cloud` defaults to `aws`
- `app.region` defaults to `us-east-1`
- `app.env` defaults to `dev`
- `service.runtime` defaults to `node`
- `service.path` defaults to `server`
- `service.port` defaults to `3000`
- `service.health` defaults to `/health`
- service exposure defaults to `private`
- resource `type` defaults to `managed`
- resource `size` defaults to `small`
- the MVP allows exactly one service

## Example output

For [examples/orders.sai](/Users/sleepercode/Documents/sai/SAI/examples/orders.sai), `sai plan --path examples/orders.sai --json` would produce a shape like:

```json
{
  "ir": {
    "application": {
      "name": "orders",
      "slug": "orders",
      "cloud": "aws",
      "region": "us-east-1",
      "users": 5000,
      "budget_usd": 75,
      "env": "prod"
    },
    "service": {
      "name": "api",
      "runtime": "node",
      "path": "server",
      "port": 3000,
      "exposure": "public_http",
      "health_check_path": "/health",
      "connects": ["postgres"],
      "scale_hint": "balanced"
    },
    "resources": [
      {
        "name": "postgres",
        "kind": "database",
        "type": "managed",
        "size": "small"
      }
    ]
  },
  "plan": {
    "profile": "balanced-web",
    "infra_class": "managed-small",
    "min_instances": 1,
    "max_instances": 3,
    "estimated_monthly_usd": 75
  }
}
```

## CLI examples

```bash
go run . init
go run . validate --path examples/orders.sai
go run . validate --path examples/orders.sai --json
go run . plan --path examples/orders.sai
go run . plan --path examples/orders.sai --json
```

## Notes

- `go` is not installed in the current environment, so this code has not been compiled here.
- The `deploy`, `logs`, and `rollback` commands are intentionally stubbed to preserve the architecture while stopping at the requested first compiler slice.
