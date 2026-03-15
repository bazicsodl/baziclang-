import { NextResponse } from "next/server";
import { prisma } from "@/lib/db";
import { createSession, verifyPassword } from "@/lib/auth";
import { loginSchema } from "@/lib/validators";

export async function POST(req: Request) {
  const json = await req.json();
  const parsed = loginSchema.safeParse(json);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: "Invalid input" }, { status: 400 });
  }

  const user = await prisma.user.findUnique({
    where: { email: parsed.data.email }
  });
  if (!user) {
    return NextResponse.json({ ok: false, error: "Invalid credentials" }, { status: 401 });
  }

  const valid = await verifyPassword(parsed.data.password, user.passwordHash);
  if (!valid) {
    return NextResponse.json({ ok: false, error: "Invalid credentials" }, { status: 401 });
  }

  await createSession(user.id);

  return NextResponse.json({ ok: true, user: { id: user.id, email: user.email, name: user.name } });
}
