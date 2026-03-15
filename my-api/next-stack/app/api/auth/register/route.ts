import { NextResponse } from "next/server";
import { prisma } from "@/lib/db";
import { createSession, hashPassword } from "@/lib/auth";
import { registerSchema } from "@/lib/validators";

export async function POST(req: Request) {
  const json = await req.json();
  const parsed = registerSchema.safeParse(json);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: "Invalid input" }, { status: 400 });
  }

  const existing = await prisma.user.findUnique({
    where: { email: parsed.data.email }
  });
  if (existing) {
    return NextResponse.json({ ok: false, error: "Email already in use" }, { status: 409 });
  }

  const passwordHash = await hashPassword(parsed.data.password);
  const user = await prisma.user.create({
    data: {
      email: parsed.data.email,
      name: parsed.data.name ?? null,
      passwordHash
    }
  });

  await createSession(user.id);

  return NextResponse.json({ ok: true, user: { id: user.id, email: user.email, name: user.name } });
}
