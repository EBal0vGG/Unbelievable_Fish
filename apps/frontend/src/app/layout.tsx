import type { ReactNode } from "react";
import type { Metadata } from "next";

import { Providers } from "@/app/providers";
import { SiteHeader } from "@/widgets/header/site-header";

import "./globals.css";

export const metadata: Metadata = {
  title: "Unbelievable Fish",
  description: "Оптовые продукты, лоты и аукционы.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <html lang="ru">
      <body>
        <Providers>
          <div className="app-shell">
            <SiteHeader />
            <main className="page-shell">{children}</main>
          </div>
        </Providers>
      </body>
    </html>
  );
}
