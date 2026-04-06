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
            <p className="eyebrow">Профиль</p>
            <h1>{mode === "login" ? "Вход в систему" : "Регистрация компании"}</h1>
            <p className="muted">Укажите данные компании и пользователя для работы в системе.</p>
          </div>

          <form className="stack-md" onSubmit={onSubmit}>
            <Field label="Компания" error={form.formState.errors.companyId?.message}>
              <Input placeholder="north-sea-llc" {...form.register("companyId")} />
            </Field>

            <Field label="Пользователь" error={form.formState.errors.userId?.message}>
              <Input placeholder="manager-01" {...form.register("userId")} />
            </Field>

            <Field label="Роль" error={form.formState.errors.role?.message}>
              <Select {...form.register("role")}>
                <option value="user">Обычный пользователь</option>
                <option value="admin">Администратор</option>
              </Select>
            </Field>

            <div className="inline-actions">
              <Button type="submit">{mode === "login" ? "Войти" : "Создать профиль"}</Button>
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
