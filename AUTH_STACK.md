# Bazic Auth Stack

## Generate Auth Schema
```powershell
.\bin\bazic.exe model auth --path bazic.auth.schema.json
.\bin\bazic.exe model migrate --schema bazic.auth.schema.json --migrations migrations --snapshot .bazic\schema.snapshot.auth.json --name auth_init
.\bin\bazic.exe migrate apply --dir migrations --driver sqlite --dsn app.db
```

## Use Auth Helpers
```bazic
let token = auth_session_create("app.db", "user-id", 14);
let user = auth_session_user("app.db", req, "bazic_session");
```

## JWT Example
```bazic
let header = "{\"alg\":\"HS256\",\"typ\":\"JWT\"}";
let payload = "{\"sub\":\"user-id\",\"role\":\"admin\"}";
let signed = jwt_sign_hs256(header, payload, "super-secret");
if signed.is_ok {
    let okv = jwt_verify_hs256(signed.value, "super-secret");
    println(okv.value);
}
```
