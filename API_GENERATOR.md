# Bazic API Generator

Generate Bazic handler functions from routes + models.

## Usage
```powershell
.\bin\bazic.exe api --routes .\examples\apps\authstack\main.bz --models .\models.bz --out handlers.bz
```

This creates handler functions for list/get/create based on the route naming convention.
