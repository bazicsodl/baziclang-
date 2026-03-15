import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Bazic Next Stack",
  description: "Full-stack reference app with auth, CRUD, and Prisma."
};

export default function RootLayout({
  children
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        {children}
      </body>
    </html>
  );
}
