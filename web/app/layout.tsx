import type { Metadata } from "next";
import { IBM_Plex_Sans, IBM_Plex_Mono } from "next/font/google";
import { Sidebar } from "@/components/Sidebar";
import { getMetrics } from "@/lib/api-client/client";
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

export default async function RootLayout({ children }: LayoutProps<"/">) {
  // Best-effort: the sidebar's connector badge is a convenience, not load-
  // bearing, so a metrics fetch failure (backend not running, tenant with no
  // data yet) just omits the badge rather than breaking every page's layout.
  let connectorTarget: string | undefined;
  let connectorVersion: string | undefined;
  try {
    const snap = await getMetrics();
    connectorTarget = snap.connector_target;
    connectorVersion = snap.connector_version;
  } catch {
    // omit the badge
  }

  return (
    <html lang="en" className={`${plexSans.variable} ${plexMono.variable} h-full antialiased`}>
      <body className="flex h-full min-h-screen">
        <Sidebar connectorTarget={connectorTarget} connectorVersion={connectorVersion} />
        <div className="flex min-w-0 flex-1 flex-col overflow-y-auto px-[34px] pt-[26px] pb-8">{children}</div>
      </body>
    </html>
  );
}
