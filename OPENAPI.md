# Bazic OpenAPI Generator

Generate an OpenAPI 3.0 spec directly from Bazic server routes (and optionally model structs).

## Usage
```powershell
.\bin\bazic.exe openapi --routes .\examples\apps\authstack\main.bz --models .\models.bz --out openapi.json --title "Bazic API" --version v1
```

## Notes
- Routes are detected using the same `fn get_.../post_...` convention as `http_serve_app`.
- `--models` is optional. When provided, structs in `models.bz` are converted into OpenAPI schemas.
- When a route path matches a model name or its plural, request/response bodies are generated automatically.
