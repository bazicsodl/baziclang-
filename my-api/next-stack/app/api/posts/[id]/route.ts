import { NextResponse } from "next/server";
import { prisma } from "@/lib/db";
import { postSchema } from "@/lib/validators";
import { getSessionUser } from "@/lib/auth";

export async function GET(
  _req: Request,
  { params }: { params: { id: string } }
) {
  const id = Number(params.id);
  if (!Number.isFinite(id)) {
    return NextResponse.json({ ok: false, error: "Invalid id" }, { status: 400 });
  }

  const post = await prisma.post.findUnique({
    where: { id },
    include: { author: true }
  });

  if (!post) {
    return NextResponse.json({ ok: false, error: "Not found" }, { status: 404 });
  }

  return NextResponse.json({
    ok: true,
    post: {
      id: post.id,
      title: post.title,
      content: post.content,
      published: post.published,
      createdAt: post.createdAt.toISOString(),
      author: {
        id: post.author.id,
        email: post.author.email,
        name: post.author.name
      }
    }
  });
}

export async function PATCH(
  req: Request,
  { params }: { params: { id: string } }
) {
  const id = Number(params.id);
  if (!Number.isFinite(id)) {
    return NextResponse.json({ ok: false, error: "Invalid id" }, { status: 400 });
  }

  const user = await getSessionUser();
  if (!user) {
    return NextResponse.json({ ok: false, error: "Unauthorized" }, { status: 401 });
  }

  const json = await req.json();
  const parsed = postSchema.partial().safeParse(json);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: "Invalid input" }, { status: 400 });
  }

  const existing = await prisma.post.findUnique({ where: { id } });
  if (!existing) {
    return NextResponse.json({ ok: false, error: "Not found" }, { status: 404 });
  }
  if (existing.authorId !== user.id) {
    return NextResponse.json({ ok: false, error: "Forbidden" }, { status: 403 });
  }

  const post = await prisma.post.update({
    where: { id },
    data: {
      title: parsed.data.title ?? existing.title,
      content: parsed.data.content ?? existing.content
    },
    include: { author: true }
  });

  return NextResponse.json({
    ok: true,
    post: {
      id: post.id,
      title: post.title,
      content: post.content,
      published: post.published,
      createdAt: post.createdAt.toISOString(),
      author: {
        id: post.author.id,
        email: post.author.email,
        name: post.author.name
      }
    }
  });
}

export async function DELETE(
  _req: Request,
  { params }: { params: { id: string } }
) {
  const id = Number(params.id);
  if (!Number.isFinite(id)) {
    return NextResponse.json({ ok: false, error: "Invalid id" }, { status: 400 });
  }

  const user = await getSessionUser();
  if (!user) {
    return NextResponse.json({ ok: false, error: "Unauthorized" }, { status: 401 });
  }

  const existing = await prisma.post.findUnique({ where: { id } });
  if (!existing) {
    return NextResponse.json({ ok: false, error: "Not found" }, { status: 404 });
  }
  if (existing.authorId !== user.id) {
    return NextResponse.json({ ok: false, error: "Forbidden" }, { status: 403 });
  }

  await prisma.post.delete({ where: { id } });

  return NextResponse.json({ ok: true });
}
