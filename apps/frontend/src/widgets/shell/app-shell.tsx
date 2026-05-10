"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { useAuth } from "@/entities/session/model/auth-context";
import { getMainInterfaceMode, isAdminSession, isSellerSession } from "@/shared/lib/access";
import { cn } from "@/shared/lib/cn";
import { displayCompany, displayPerson, initialsFromName } from "@/shared/lib/display";
import { roleLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";

const baseNavItems = [
  { href: "/", label: "Биржа", code: "EX" },
  { href: "/catalog", label: "Каталог", code: "CA" },
  { href: "/products", label: "Продукты", code: "PR" },
  { href: "/lots", label: "Лоты", code: "LT" },
  { href: "/auctions", label: "Торги", code: "TR" },
  { href: "/deals", label: "Сделки", code: "DL" },
  { href: "/create/lot", label: "Новый лот", code: "NL" },
  { href: "/me", label: "Профиль", code: "ME" },
];

const sectionMeta: Record<string, { title: string; subtitle: string }> = {
  "/": { title: "Операционная панель", subtitle: "Обзор рынка, торгов и сделок" },
  "/catalog": { title: "Каталог", subtitle: "Справочник рыбных активов" },
  "/products": { title: "Продукты", subtitle: "Позиции вашей компании" },
  "/lots": { title: "Лоты", subtitle: "Партии для публикации и торгов" },
  "/auctions": { title: "Торги", subtitle: "Биржевая лента аукционов" },
  "/deals": { title: "Сделки", subtitle: "Контракты после торгов" },
  "/create/lot": { title: "Новый лот", subtitle: "Сборка продукта, партии и торгов" },
  "/me": { title: "Профиль", subtitle: "Компания, доступы и активность" },
  "/create/fish": { title: "Новая рыба", subtitle: "Администрирование каталога" },
};

function routeMeta(pathname: string) {
  const exact = sectionMeta[pathname];
  if (exact) {
    return exact;
  }
  const match = Object.entries(sectionMeta)
    .filter(([href]) => href !== "/" && pathname.startsWith(`${href}/`))
    .sort((left, right) => right[0].length - left[0].length)[0];

  return match?.[1] ?? sectionMeta["/"];
}

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { session, logout } = useAuth();
  const interfaceMode = getMainInterfaceMode(session);
  const canManageFish = isAdminSession(session);
  const canCreateSupply = isSellerSession(session);
  const meta = routeMeta(pathname);
  const personName = displayPerson(session);
  const companyLabel = session?.companyId ? displayCompany(session.companyId) : roleLabels[session?.role ?? "buyer_seller"];
  const visibleNavItems = baseNavItems.filter((item) => {
    if (interfaceMode === "buyer" && (item.href === "/products" || item.href === "/lots")) {
      return false;
    }
    if (item.href === "/deals" && !session?.companyId) {
      return false;
    }
    if (item.href === "/create/lot" && !canCreateSupply) {
      return false;
    }
    return true;
  });
  const navItems = canManageFish
    ? [
        ...visibleNavItems.slice(0, 5),
        { href: "/create/fish", label: "Новая рыба", code: "NF" },
        ...visibleNavItems.slice(5),
      ]
    : visibleNavItems;

  return (
    <div className="marketplace-shell">
      <header className="floating-header">
        <Link className="floating-brand" href="/" prefetch={false}>
          <span className="floating-brand-mark">
            <Image alt="Рыбная биржа" height={36} priority src="/fish-exchange-logo.svg" width={36} />
          </span>
          <span>
            <strong>Рыбная биржа</strong>
            <small>B2B seafood exchange</small>
          </span>
        </Link>

        <nav aria-label="Основная навигация" className="floating-nav">
          {navItems.map((item) => {
            const isActive = pathname === item.href || (item.href !== "/" && pathname.startsWith(`${item.href}/`));

            return (
              <Link
                aria-current={isActive ? "page" : undefined}
                className={cn("floating-nav-link", isActive && "floating-nav-link-active")}
                href={item.href}
                key={item.href}
                prefetch={false}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="floating-actions">
          {canCreateSupply ? (
            <Link className={buttonStyles({ variant: "primary", size: "sm" })} href="/create/lot" prefetch={false}>
              Новый лот
            </Link>
          ) : null}
          {session ? (
            <>
              <Link
                className="user-badge"
                href="/me"
                prefetch={false}
                title={session.companyId ? `Компания: ${session.companyId}` : personName}
              >
                <span>{initialsFromName(personName)}</span>
                <strong>{personName}</strong>
                <small>{companyLabel} · {roleLabels[session.role]}</small>
              </Link>
              <button className={buttonStyles({ variant: "ghost", size: "sm" })} onClick={logout} type="button">
                Выйти
              </button>
            </>
          ) : (
            <>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login" prefetch={false}>
                Вход
              </Link>
              <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/register" prefetch={false}>
                Регистрация
              </Link>
            </>
          )}
        </div>
      </header>

      <div className="floating-context">
        <p>{meta.subtitle}</p>
        <strong>{meta.title}</strong>
      </div>

      <main className="marketplace-content">{children}</main>
    </div>
  );
}
