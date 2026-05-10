import type { ReactNode } from "react";
import type { Metadata } from "next";

import { Providers } from "@/app/providers";
import { AppShell } from "@/widgets/shell/app-shell";

import "./globals.css";

export const metadata: Metadata = {
  title: "Unbelievable Fish Exchange",
  description: "B2B marketplace для рыбной продукции, торгов и сделок.",
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
          <AppShell>{children}</AppShell>
        </Providers>
      </body>
    </html>
  );
}
