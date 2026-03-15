import { z } from "zod";

export const registerSchema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(80).optional(),
  password: z.string().min(8).max(128)
});

export const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8).max(128)
});

export const postSchema = z.object({
  title: z.string().min(2).max(120),
  content: z.string().min(1).max(2000)
});
