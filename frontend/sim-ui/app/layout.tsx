import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Smart Home Simulation",
  description: "UI simulation",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru">
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
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
