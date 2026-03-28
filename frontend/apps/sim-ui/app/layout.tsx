import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Smart Home Simulation",
  description: "UI simulation",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru">
      <body className="antialiased">
        <div
          className="fixed inset-0 -z-10 pointer-events-none"
          style={{
            background:
              "radial-gradient(1200px 600px at 20% 0%, rgba(120,119,198,0.16), transparent 60%), radial-gradient(900px 500px at 80% 10%, rgba(56,189,248,0.10), transparent 60%), #0b0c0f",
          }}
        />

        <div className="relative z-10 min-h-screen">{children}</div>
      </body>
    </html>
  );
}
