import ClientApp from "@/components/ClientApp";

export default function Page() {
  return (
    <main className="page">
      <header className="hero">
        <div>
          <p className="eyebrow">Bazic Language • Next.js Stack</p>
          <h1>Models, database, auth, and CRUD — ready to ship.</h1>
          <p className="subtext">
            This reference app includes Prisma models, SQLite config, session auth,
            password hashing, and full REST endpoints with a functional UI.
          </p>
        </div>
        <div className="hero-card">
          <h3>What you get</h3>
          <ul>
            <li>Secure auth + sessions</li>
            <li>REST API for posts</li>
            <li>Prisma models & migrations</li>
            <li>Client UI that calls the API</li>
          </ul>
        </div>
      </header>
      <ClientApp />
      <footer className="footer">Built for Bazic — extend freely.</footer>
    </main>
  );
}
