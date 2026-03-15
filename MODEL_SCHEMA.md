# Bazic Model Schema

This is the Bazic schema format for defining models and generating migrations.

## File
Default: `bazic.schema.json`

## Example
```json
{
  "version": 1,
  "database": { "driver": "sqlite", "dsn": "app.db" },
  "models": [
    {
      "name": "User",
      "table": "users",
      "fields": [
        { "name": "id", "type": "int", "pk": true, "auto": true },
        { "name": "email", "type": "string", "unique": true },
        { "name": "name", "type": "string", "optional": true },
        { "name": "created_at", "type": "time", "default": "raw:CURRENT_TIMESTAMP" }
      ],
      "indexes": [
        { "name": "idx_users_email", "fields": ["email"], "unique": true }
      ]
    }
  ]
}
```

## Field Types
- `int`, `float`, `bool`, `string`, `text`, `time`, `json`

## Field Constraints
Optional fields you can add to any `field`:
- `min_len`, `max_len` for strings
- `min`, `max` for ints
- `min_f`, `max_f` for floats
- `enum` array of allowed string values
- `join` to generate nested join helpers (format: `Model.field`)

## Defaults
- Use `raw:` prefix for DB expressions: `raw:CURRENT_TIMESTAMP`

## CLI
```powershell
# Create a schema
.\bin\bazic.exe model init

# Create auth schema (User + Session)
.\bin\bazic.exe model auth

# Generate OpenAPI from Bazic routes
.\bin\bazic.exe openapi --routes .\examples\apps\authstack\main.bz --models .\models.bz --out openapi.json

# Generate Bazic structs
.\bin\bazic.exe model generate --schema bazic.schema.json --out models.bz

# Generate migration from schema changes
.\bin\bazic.exe model migrate --schema bazic.schema.json --migrations migrations --snapshot .bazic\schema.snapshot.json --name init

# Apply migrations
.\bin\bazic.exe migrate apply --dir migrations --driver sqlite --dsn app.db
```
