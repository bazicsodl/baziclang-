# Bazic Next Stack

A production-ready Next.js + Prisma + SQLite stack with auth, sessions, and CRUD APIs.

## Quickstart
```powershell
cd C:\Users\Ipeh\Documents\baziclang\my-api\next-stack
npm install
copy .env.example .env
npm run prisma:migrate -- --name init
npm run dev
```

## API
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/posts`
- `POST /api/posts`
- `GET /api/posts/:id`
- `PATCH /api/posts/:id`
- `DELETE /api/posts/:id`
- `GET /api/health`

## Notes
- Sessions are stored in the database and set as HTTP-only cookies.
- Passwords are hashed with bcrypt.
- Prisma schema is in `prisma/schema.prisma`.
