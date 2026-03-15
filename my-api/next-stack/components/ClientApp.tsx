"use client";

import { useEffect, useState } from "react";

type User = {
  id: number;
  email: string;
  name: string | null;
};

type Post = {
  id: number;
  title: string;
  content: string;
  published: boolean;
  author: User;
  createdAt: string;
};

export default function ClientApp() {
  const [user, setUser] = useState<User | null>(null);
  const [posts, setPosts] = useState<Post[]>([]);
  const [status, setStatus] = useState<string>("");
  const [form, setForm] = useState({ email: "", name: "", password: "" });
  const [postForm, setPostForm] = useState({ title: "", content: "" });

  const loadMe = async () => {
    const res = await fetch("/api/auth/me", { credentials: "include" });
    const data = await res.json();
    setUser(data.user ?? null);
  };

  const loadPosts = async () => {
    const res = await fetch("/api/posts", { credentials: "include" });
    const data = await res.json();
    setPosts(data.posts ?? []);
  };

  useEffect(() => {
    void loadMe();
    void loadPosts();
  }, []);

  const register = async () => {
    setStatus("");
    const res = await fetch("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(form)
    });
    const data = await res.json();
    if (!res.ok) {
      setStatus(data.error ?? "Registration failed");
      return;
    }
    setStatus("Registered and signed in.");
    setForm({ email: "", name: "", password: "" });
    await loadMe();
  };

  const login = async () => {
    setStatus("");
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ email: form.email, password: form.password })
    });
    const data = await res.json();
    if (!res.ok) {
      setStatus(data.error ?? "Login failed");
      return;
    }
    setStatus("Signed in.");
    await loadMe();
  };

  const logout = async () => {
    await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
    setUser(null);
    setStatus("Signed out.");
  };

  const createPost = async () => {
    setStatus("");
    const res = await fetch("/api/posts", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(postForm)
    });
    const data = await res.json();
    if (!res.ok) {
      setStatus(data.error ?? "Create failed");
      return;
    }
    setPostForm({ title: "", content: "" });
    setStatus("Post created.");
    await loadPosts();
  };

  return (
    <section className="section">
      <div className="grid">
        <div className="card">
          <h3>Auth</h3>
          <div className="grid">
            <div>
              <label>Email</label>
              <input
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                placeholder="you@bazic.dev"
              />
            </div>
            <div>
              <label>Name</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="Optional"
              />
            </div>
            <div>
              <label>Password</label>
              <input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                placeholder="********"
              />
            </div>
          </div>
          <div className="grid">
            <button onClick={register}>Register</button>
            <button className="secondary" onClick={login}>Login</button>
            <button className="secondary" onClick={logout}>Logout</button>
          </div>
          {user ? (
            <p className="message">Signed in as {user.email}</p>
          ) : (
            <p className="message">Not signed in</p>
          )}
        </div>

        <div className="card">
          <h3>Create Post</h3>
          <div className="grid">
            <div>
              <label>Title</label>
              <input
                value={postForm.title}
                onChange={(e) => setPostForm({ ...postForm, title: e.target.value })}
                placeholder="Launch update"
              />
            </div>
            <div>
              <label>Content</label>
              <textarea
                rows={4}
                value={postForm.content}
                onChange={(e) => setPostForm({ ...postForm, content: e.target.value })}
                placeholder="Write your update..."
              />
            </div>
          </div>
          <button onClick={createPost}>Publish</button>
        </div>
      </div>

      <div className="card">
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h3>Posts</h3>
          <button className="secondary" onClick={loadPosts}>Refresh</button>
        </div>
        <div className="list">
          {posts.map((post) => (
            <div key={post.id} className="card">
              <div className="badge">#{post.id}</div>
              <h4>{post.title}</h4>
              <p className="subtext">{post.content}</p>
              <p className="subtext">By {post.author?.email ?? "Unknown"}</p>
            </div>
          ))}
          {posts.length === 0 && <p className="subtext">No posts yet.</p>}
        </div>
      </div>

      {status && <p className="message">{status}</p>}
    </section>
  );
}
