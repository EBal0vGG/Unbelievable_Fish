"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";

const schema = z.object({
  companyId: z.string().min(2, "Укажите companyId"),
  userId: z.string().min(2, "Укажите userId"),
  role: z.enum(["admin", "user"]),
});

type AuthValues = z.infer<typeof schema>;

export function AuthForm({ mode }: { mode: "login" | "register" }) {
  const router = useRouter();
  const { saveSession } = useAuth();
  const form = useForm<AuthValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      companyId: "",
      userId: "",
      role: "user",
    },
  });

  const onSubmit = form.handleSubmit((values) => {
    saveSession(values.companyId, values.userId, values.role, mode);
    router.push("/");
  });

  return (
    <div className="auth-shell">
      <Card className="auth-card">
        <div className="stack-lg">
          <div>
            <p className="eyebrow">MVP Auth</p>
            <h1>{mode === "login" ? "Вход в биржу" : "Регистрация компании"}</h1>
            <p className="muted">
              Временный контекст пользователя хранится во frontend storage и пробрасывается в
              `X-Company-ID` / `X-User-ID`.
            </p>
          </div>

          {mode === "register" ? (
            <Notice tone="warning" title="Временная регистрация">
              Реального backend-flow регистрации сейчас нет. Экран готов для будущей интеграции и
              пока сохраняет только локальный контекст пользователя.
            </Notice>
          ) : null}

          <form className="stack-md" onSubmit={onSubmit}>
            <Field label="Company ID" error={form.formState.errors.companyId?.message}>
              <Input placeholder="north-sea-llc" {...form.register("companyId")} />
            </Field>

            <Field label="User ID" error={form.formState.errors.userId?.message}>
              <Input placeholder="manager-01" {...form.register("userId")} />
            </Field>

            <Field label="Роль" error={form.formState.errors.role?.message}>
              <Select {...form.register("role")}>
                <option value="user">Обычный пользователь</option>
                <option value="admin">Администратор</option>
              </Select>
            </Field>

            <div className="inline-actions">
              <Button type="submit">{mode === "login" ? "Войти" : "Сохранить контекст"}</Button>
              <Link
                className={buttonStyles({ variant: "ghost" })}
                href={mode === "login" ? "/register" : "/login"}
              >
                {mode === "login" ? "Регистрация" : "Назад ко входу"}
              </Link>
            </div>
          </form>
        </div>
      </Card>
    </div>
  );
}
