"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { useAuth } from "@/entities/session/model/auth-context";
import { isAdminSession } from "@/shared/lib/access";
import { cn } from "@/shared/lib/cn";
import { buttonStyles } from "@/shared/ui/button";

const navItems = [
  { href: "/", label: "Главная" },
  { href: "/catalog", label: "Каталог" },
  { href: "/lots", label: "Лоты" },
  { href: "/auctions", label: "Аукционы" },
  { href: "/create/lot", label: "Создать лот" },
  { href: "/create/auction", label: "Создать аукцион" },
  { href: "/me", label: "Мой контекст" },
];

export function SiteHeader() {
  const pathname = usePathname();
  const { session, logout } = useAuth();
  const canManageFish = isAdminSession(session);
  const visibleNavItems = canManageFish
    ? [...navItems.slice(0, 4), { href: "/create/fish", label: "Создать рыбу" }, ...navItems.slice(4)]
    : navItems;

  return (
    <header className="site-header">
      <div className="brand-block">
        <Link href="/" className="brand-mark">
          <span>UF</span>
        </Link>
        <div>
          <p className="brand-title">Fish Exchange MVP</p>
          <p className="brand-subtitle">B2B marketplace / lots / auctions</p>
        </div>
      </div>

      <nav className="main-nav">
        {visibleNavItems.map((item) => (
          <Link
            key={item.href}
            className={cn("nav-link", pathname === item.href && "nav-link-active")}
            href={item.href}
          >
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="header-actions">
        {session ? (
          <>
            <Link className="session-chip" href="/me">
              <span>{session.companyId}</span>
              <span>{session.userId} · {session.role}</span>
            </Link>
            <button className={buttonStyles({ variant: "ghost", size: "sm" })} onClick={logout} type="button">
              Выйти
            </button>
          </>
        ) : (
          <>
            <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login">
              Вход
            </Link>
            <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/register">
              Регистрация
            </Link>
          </>
        )}
      </div>
    </header>
  );
}
