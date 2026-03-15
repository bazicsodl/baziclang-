# Bazic JWT Stack

This example shows JWT issuance + verification plus CSRF token generation.

## Run
```powershell
.\bin\bazic.exe run examples\apps\jwtstack\main.bz
```

## Routes
- `POST /login`
- `GET /me`
- `GET /csrf`

## Example
```powershell
# Login (demo credentials)
$body = '{"email":"admin@bazic.dev","password":"secret"}'
$token = (Invoke-RestMethod -Method Post -Uri http://localhost:8080/login -Body $body -ContentType 'application/json').token

# Verify token
Invoke-RestMethod -Method Get -Uri http://localhost:8080/me -Headers @{ Authorization = "Bearer $token" }

# CSRF token
Invoke-RestMethod -Method Get -Uri http://localhost:8080/csrf
```

## Notes
- This example uses an in-memory demo user.
- Replace `SECRET` in `examples/apps/jwtstack/main.bz` for production.
