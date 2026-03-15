import { NextResponse } from "next/server";
import { getSessionUser } from "@/lib/auth";

export async function GET() {
  const user = await getSessionUser();
  if (!user) {
    return NextResponse.json({ ok: true, user: null });
  }
  return NextResponse.json({ ok: true, user: { id: user.id, email: user.email, name: user.name } });
}
