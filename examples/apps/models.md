# Bazic Model + Migration Example

## 1) Initialize schema
```powershell
.\bin\bazic.exe model init
```

## 2) Edit `bazic.schema.json`
Add models and fields.

## 3) Generate Bazic structs
```powershell
.\bin\bazic.exe model generate --schema bazic.schema.json --out models.bz
```

## 4) Create migration
```powershell
.\bin\bazic.exe model migrate --schema bazic.schema.json --migrations migrations --snapshot .bazic\schema.snapshot.json --name init
```

## 5) Apply migration
```powershell
.\bin\bazic.exe migrate apply --dir migrations --driver sqlite --dsn app.db
```
