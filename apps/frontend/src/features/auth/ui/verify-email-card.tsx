"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { ApiError } from "@/shared/api/http-client";
import { verifyEmail } from "@/shared/api/identity-service";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";

type VerifyState = "loading" | "success" | "error";

export function VerifyEmailCard({ token }: { token?: string }) {
  const [state, setState] = useState<VerifyState>("loading");
  const [message, setMessage] = useState("Подтверждаем email...");

  useEffect(() => {
    let ignore = false;
    const run = async () => {
      if (!token) {
        setState("error");
        setMessage("Ссылка подтверждения некорректна.");
        return;
      }
      try {
        const result = await verifyEmail(token);
        if (ignore) {
          return;
        }
        setState("success");
        setMessage(
          result.already_verified
            ? "Email уже был подтвержден. Теперь можно войти."
            : "Email подтвержден. Теперь можно войти.",
        );
      } catch (error) {
        if (ignore) {
          return;
        }
        setState("error");
        setMessage(error instanceof ApiError ? mapVerifyError(error) : "Не удалось подтвердить email.");
      }
    };
    void run();
    return () => {
      ignore = true;
    };
  }, [token]);

  return (
    <div className="auth-shell">
      <Card className="auth-card">
        <div className="stack-lg">
          <div>
            <h1>Подтверждение email</h1>
            <p className="muted">{state === "loading" ? "Пожалуйста, подождите." : "Рыбная биржа"}</p>
          </div>

          <Notice tone={state === "success" ? "success" : state === "error" ? "warning" : "info"}>
            {message}
          </Notice>

          <div className="inline-actions">
            <Link className={buttonStyles({})} href="/login">
              Войти
            </Link>
            {state === "error" ? (
              <Link className={buttonStyles({ variant: "ghost" })} href="/login">
                Отправить письмо повторно
              </Link>
            ) : null}
          </div>
        </div>
      </Card>
    </div>
  );
}

function mapVerifyError(error: ApiError): string {
  switch (error.code) {
    case "VERIFICATION_TOKEN_EXPIRED":
      return "Срок действия ссылки истек. Отправьте письмо подтверждения повторно.";
    case "VERIFICATION_TOKEN_USED":
      return "Эта ссылка уже была использована.";
    case "VERIFICATION_TOKEN_REQUIRED":
    case "VERIFICATION_TOKEN_INVALID":
      return "Ссылка подтверждения некорректна.";
    default:
      return error.message;
  }
}
