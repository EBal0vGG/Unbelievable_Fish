"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { registerCompany, registerUser } from "@/shared/api/identity-service";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";

const loginSchema = z.object({
  login: z.string().min(2, "Укажите логин или email"),
  password: z.string().min(1, "Укажите пароль"),
});

const companySchema = z.object({
  name: z.string().min(2, "Укажите название компании"),
  inn: z.string().min(10, "Укажите корректный ИНН"),
  ogrn: z.string().min(13, "Укажите корректный ОГРН"),
});

const registerUserSchema = z.object({
  name: z.string().min(2, "Укажите имя пользователя"),
  role: z.enum(["admin", "seller", "buyer"]),
  login: z.string().min(2, "Укажите логин или email"),
  password: z.string().min(1, "Укажите пароль"),
});

type LoginValues = z.infer<typeof loginSchema>;
type CompanyValues = z.infer<typeof companySchema>;
type RegisterUserValues = z.infer<typeof registerUserSchema>;

export function AuthForm({
  mode,
  nextPath = "/",
}: {
  mode: "login" | "register";
  nextPath?: string;
}) {
  const router = useRouter();
  const { login } = useAuth();
  const [authError, setAuthError] = useState<string | null>(null);
  const [createdCompany, setCreatedCompany] = useState<{ id: string; name: string } | null>(null);
  const [companyMessage, setCompanyMessage] = useState<string | null>(null);
  const isCompanyRegistered = Boolean(createdCompany);

  const loginForm = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      login: "",
      password: "",
    },
  });
  const companyForm = useForm<CompanyValues>({
    resolver: zodResolver(companySchema),
    defaultValues: {
      name: "",
      inn: "",
      ogrn: "",
    },
  });
  const registerUserForm = useForm<RegisterUserValues>({
    resolver: zodResolver(registerUserSchema),
    defaultValues: {
      name: "",
      role: "seller",
      login: "",
      password: "",
    },
  });

  const submitLogin = loginForm.handleSubmit(async (values) => {
    setAuthError(null);
    try {
      await login(values.login, values.password);
      router.push(nextPath);
    } catch (error) {
      setAuthError(error instanceof ApiError ? error.message : "Не удалось войти в систему.");
    }
  });

  const submitCompany = companyForm.handleSubmit(async (values) => {
    setAuthError(null);
    try {
      const company = await registerCompany(values);
      setCreatedCompany({ id: company.id, name: company.name });
      setCompanyMessage(`Компания ${company.name} зарегистрирована. Теперь создайте пользователя.`);
    } catch (error) {
      setAuthError(error instanceof ApiError ? error.message : "Не удалось зарегистрировать компанию.");
    }
  });

  const submitUser = registerUserForm.handleSubmit(async (values) => {
    if (!createdCompany) {
      setAuthError("Сначала зарегистрируйте компанию.");
      return;
    }

    setAuthError(null);
    try {
      await registerUser({
        companyId: createdCompany.id,
        name: values.name,
        role: values.role,
        login: values.login,
        password: values.password,
      });
      await login(values.login, values.password);
      router.push(nextPath);
    } catch (error) {
      setAuthError(error instanceof ApiError ? error.message : "Не удалось завершить регистрацию.");
    }
  });

  if (mode === "login") {
    return (
      <div className="auth-shell">
        <Card className="auth-card">
          <div className="stack-lg">
          <div>
              <h1>Вход в систему</h1>
              <p className="muted">Введите логин и пароль, чтобы войти в систему.</p>
            </div>

            {authError ? (
              <Notice tone="warning" title="Ошибка входа">
                {authError}
              </Notice>
            ) : null}

            <form className="stack-md" onSubmit={submitLogin}>
              <Field label="Логин или email" error={loginForm.formState.errors.login?.message}>
                <Input placeholder="manager@north-sea.ru" {...loginForm.register("login")} />
              </Field>

              <Field label="Пароль" error={loginForm.formState.errors.password?.message}>
                <Input type="password" {...loginForm.register("password")} />
              </Field>

              <div className="inline-actions">
                <Button disabled={loginForm.formState.isSubmitting} type="submit">
                  {loginForm.formState.isSubmitting ? "Входим..." : "Войти"}
                </Button>
                <Link className={buttonStyles({ variant: "ghost" })} href="/register">
                  Регистрация
                </Link>
              </div>
            </form>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="auth-shell">
      <Card className="auth-card">
        <div className="stack-lg">
          <div>
            <h1>Регистрация компании и пользователя</h1>
            <p className="muted">Сначала зарегистрируйте компанию, затем создайте пользователя.</p>
          </div>

          {authError ? (
            <Notice tone="warning" title="Ошибка регистрации">
              {authError}
            </Notice>
          ) : null}

          {companyMessage ? (
            <Notice tone="success" title="Компания зарегистрирована">
              {companyMessage}
            </Notice>
          ) : null}

          <form className="stack-md" onSubmit={submitCompany}>
            <Field label="Название компании" error={companyForm.formState.errors.name?.message}>
              <Input disabled={isCompanyRegistered} placeholder="North Sea LLC" {...companyForm.register("name")} />
            </Field>

            <Field label="ИНН" error={companyForm.formState.errors.inn?.message}>
              <Input disabled={isCompanyRegistered} placeholder="1234567890" {...companyForm.register("inn")} />
            </Field>

            <Field label="ОГРН" error={companyForm.formState.errors.ogrn?.message}>
              <Input disabled={isCompanyRegistered} placeholder="1234567890123" {...companyForm.register("ogrn")} />
            </Field>

            <div className="inline-actions">
              <Button disabled={companyForm.formState.isSubmitting || isCompanyRegistered} type="submit">
                {companyForm.formState.isSubmitting
                  ? "Сохраняем..."
                  : isCompanyRegistered
                    ? "Компания зарегистрирована"
                    : "Зарегистрировать компанию"}
              </Button>
            </div>
          </form>

          <form className="stack-md" onSubmit={submitUser}>
            <Field label="Company ID">
              <Input disabled value={createdCompany?.id ?? "Сначала зарегистрируйте компанию"} />
            </Field>

            <Field label="Имя пользователя" error={registerUserForm.formState.errors.name?.message}>
              <Input disabled={!createdCompany} placeholder="Менеджер продаж" {...registerUserForm.register("name")} />
            </Field>

            <Field label="Роль" error={registerUserForm.formState.errors.role?.message}>
              <Select disabled={!createdCompany} {...registerUserForm.register("role")}>
                <option value="seller">Seller</option>
                <option value="buyer">Buyer</option>
                <option value="admin">Admin</option>
              </Select>
            </Field>

            <Field label="Логин или email" error={registerUserForm.formState.errors.login?.message}>
              <Input disabled={!createdCompany} placeholder="manager@north-sea.ru" {...registerUserForm.register("login")} />
            </Field>

            <Field label="Пароль" error={registerUserForm.formState.errors.password?.message}>
              <Input disabled={!createdCompany} type="password" {...registerUserForm.register("password")} />
            </Field>

            <div className="inline-actions">
              <Button disabled={!createdCompany || registerUserForm.formState.isSubmitting} type="submit">
                {registerUserForm.formState.isSubmitting ? "Создаем..." : "Создать пользователя и войти"}
              </Button>
              <Link className={buttonStyles({ variant: "ghost" })} href="/login">
                Назад ко входу
              </Link>
            </div>
          </form>
        </div>
      </Card>
    </div>
  );
}
