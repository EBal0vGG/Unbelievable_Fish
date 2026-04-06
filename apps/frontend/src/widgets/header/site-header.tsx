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
  { href: "/products", label: "Продукты" },
  { href: "/lots", label: "Лоты" },
  { href: "/auctions", label: "Аукционы" },
  { href: "/create/lot", label: "Создать лот" },
  { href: "/create/auction", label: "Создать аукцион" },
  { href: "/me", label: "Мой профиль" },
];

export function SiteHeader() {
  const pathname = usePathname();
  const { session, logout } = useAuth();
  const canManageFish = isAdminSession(session);
  const visibleNavItems = canManageFish
    ? [...navItems.slice(0, 5), { href: "/create/fish", label: "Создать рыбу" }, ...navItems.slice(5)]
    : navItems;

  return (
    <header className="site-header">
      <div className="brand-block">
        <Link href="/" className="brand-mark">
          <span>UF</span>
        </Link>
        <div>
          <p className="brand-title">Unbelievable Fish</p>
          <p className="brand-subtitle">Оптовые продукты, лоты и аукционы</p>
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
