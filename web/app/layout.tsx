import type { Metadata } from "next";
import { IBM_Plex_Sans, IBM_Plex_Mono } from "next/font/google";
import "./globals.css";

const plexSans = IBM_Plex_Sans({
  variable: "--font-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
});

export const metadata: Metadata = {
  title: "SOAR Console",
  description: "SOAR lab case queue and investigation console.",
};

// Bare shell only — no Sidebar, no session/metrics fetch. Those belong to
// app/(app)/layout.tsx, which wraps every authenticated route; /login sits
// outside that group on purpose, so a signed-out screen never renders the
// app chrome (nav links to pages it can't reach yet) around it.
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${plexSans.variable} ${plexMono.variable} h-full antialiased`}>
      <body className="flex h-full min-h-screen flex-col">{children}</body>
    </html>
  );
}
