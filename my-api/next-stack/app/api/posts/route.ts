import { NextResponse } from "next/server";
import { prisma } from "@/lib/db";
import { getSessionUser } from "@/lib/auth";
import { postSchema } from "@/lib/validators";

export async function GET() {
  const posts = await prisma.post.findMany({
    orderBy: { createdAt: "desc" },
    include: { author: true }
  });

  return NextResponse.json({
    ok: true,
    posts: posts.map((post) => ({
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
    }))
  });
}

export async function POST(req: Request) {
  const user = await getSessionUser();
  if (!user) {
    return NextResponse.json({ ok: false, error: "Unauthorized" }, { status: 401 });
  }

  const json = await req.json();
  const parsed = postSchema.safeParse(json);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: "Invalid input" }, { status: 400 });
  }

  const post = await prisma.post.create({
    data: {
      title: parsed.data.title,
      content: parsed.data.content,
      published: true,
      authorId: user.id
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
