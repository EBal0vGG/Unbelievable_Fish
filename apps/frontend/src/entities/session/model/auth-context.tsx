"use client";

import { createContext, type ReactNode, useContext, useEffect, useState } from "react";

import { normalizeRole } from "@/shared/lib/access";
import { readLocalStorage, writeLocalStorage } from "@/shared/lib/storage";
import type { UserSession } from "@/shared/types/domain";

const SESSION_KEY = "uf:session-context";

interface AuthContextValue {
  session: UserSession | null;
  isReady: boolean;
  saveSession: (
    companyId: string,
    userId: string,
    role: UserSession["role"],
    mode: UserSession["mode"],
  ) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<UserSession | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const stored = readLocalStorage<(UserSession & { role?: UserSession["role"] }) | null>(SESSION_KEY, null);
    setSession(
      stored
        ? {
            ...stored,
            role: normalizeRole(stored.role),
          }
        : null,
    );
    setIsReady(true);
  }, []);

  const saveSession = (
    companyId: string,
    userId: string,
    role: UserSession["role"],
    mode: UserSession["mode"],
  ) => {
    const nextSession: UserSession = {
      companyId,
      userId,
      role,
      mode,
      updatedAt: new Date().toISOString(),
    };
    setSession(nextSession);
    writeLocalStorage(SESSION_KEY, nextSession);
  };

  const logout = () => {
    setSession(null);
    writeLocalStorage<UserSession | null>(SESSION_KEY, null);
  };

  return (
    <AuthContext.Provider value={{ session, isReady, saveSession, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }

  return context;
}
