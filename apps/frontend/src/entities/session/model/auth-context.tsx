"use client";

import { useQueryClient } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext, useEffect, useState } from "react";

import { getCurrentUser, login as loginRequest, toUserSession } from "@/shared/api/identity-service";
import { normalizeRole } from "@/shared/lib/access";
import { readLocalStorage, writeLocalStorage } from "@/shared/lib/storage";
import type { UserSession } from "@/shared/types/domain";

const SESSION_KEY = "uf:session-context";

type AuthStatus = "loading" | "authenticated" | "guest";

interface AuthContextValue {
  session: UserSession | null;
  isReady: boolean;
  status: AuthStatus;
  login: (login: string, password: string) => Promise<UserSession>;
  logout: () => void;
  refreshCurrentUser: () => Promise<UserSession | null>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function normalizeSession(session: UserSession): UserSession {
  return {
    ...session,
    role: normalizeRole(session.role),
  };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [session, setSession] = useState<UserSession | null>(null);
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const hydrate = async () => {
      const stored = readLocalStorage<UserSession | null>(SESSION_KEY, null);
      if (!stored?.accessToken) {
        setSession(null);
        setStatus("guest");
        setIsReady(true);
        writeLocalStorage<UserSession | null>(SESSION_KEY, null);
        return;
      }

      const normalized = normalizeSession(stored);
      try {
        const currentUser = await getCurrentUser(normalized);
        const nextSession = normalizeSession(
          toUserSession(normalized.accessToken, currentUser, normalized.mode),
        );
        setSession(nextSession);
        setStatus("authenticated");
        writeLocalStorage(SESSION_KEY, nextSession);
      } catch {
        setSession(null);
        setStatus("guest");
        writeLocalStorage<UserSession | null>(SESSION_KEY, null);
      } finally {
        setIsReady(true);
      }
    };

    void hydrate();
  }, []);

  const refreshCurrentUser = async (): Promise<UserSession | null> => {
    if (!session?.accessToken) {
      return null;
    }

    const currentUser = await getCurrentUser(session);
    const nextSession = normalizeSession(toUserSession(session.accessToken, currentUser, session.mode));
    setSession(nextSession);
    setStatus("authenticated");
    writeLocalStorage(SESSION_KEY, nextSession);
    return nextSession;
  };

  const login = async (loginValue: string, password: string): Promise<UserSession> => {
    const result = await loginRequest({ login: loginValue, password });
    const nextSession = normalizeSession(toUserSession(result.token, result.user, "login"));
    setSession(nextSession);
    setStatus("authenticated");
    writeLocalStorage(SESSION_KEY, nextSession);
    queryClient.clear();
    return nextSession;
  };

  const logout = () => {
    setSession(null);
    setStatus("guest");
    writeLocalStorage<UserSession | null>(SESSION_KEY, null);
    queryClient.clear();
  };

  return (
    <AuthContext.Provider value={{ session, isReady, status, login, logout, refreshCurrentUser }}>
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
