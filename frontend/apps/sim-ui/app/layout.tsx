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
              "radial-gradient(900px 520px at 12% -10%, rgba(255,255,255,0.10), transparent 60%), radial-gradient(760px 480px at 88% 5%, rgba(0,113,227,0.18), transparent 58%), linear-gradient(180deg, #030303, #111113 46%, #1d1d1f)",
          }}
        />

        <div className="relative z-10 min-h-screen">{children}</div>
      </body>
    </html>
  );
}
