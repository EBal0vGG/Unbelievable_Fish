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

const currentTermsVersion = "2026-04-24";

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
  role: z.enum(["seller", "buyer"]),
  login: z.string().min(2, "Укажите логин или email"),
  password: z.string().min(1, "Укажите пароль"),
  acceptedTerms: z.boolean().refine((value) => value, "Нужно согласиться с условиями пользования"),
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
      acceptedTerms: false,
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
      setCompanyMessage(`Компания ${company.name} выбрана. Теперь создайте пользователя.`);
    } catch (error) {
      setAuthError(error instanceof ApiError ? error.message : "Не удалось зарегистрировать компанию.");
    }
  });

  const continueWithoutCompany = () => {
    setAuthError(null);
    setCreatedCompany(null);
    setCompanyMessage("Регистрация продолжится без компании.");
  };
  const submitUser = registerUserForm.handleSubmit(async (values) => {
    setAuthError(null);
    try {
      await registerUser({
        companyId: createdCompany?.id,
        companyInn: createdCompany ? companyForm.getValues("inn") : undefined,
        companyOgrn: createdCompany ? companyForm.getValues("ogrn") : undefined,
        name: values.name,
        role: values.role,
        login: values.login,
        password: values.password,
        acceptedTerms: values.acceptedTerms,
        termsVersion: currentTermsVersion,
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
            <h1>Регистрация пользователя</h1>
            <p className="muted">Создайте пользователя сразу или при необходимости сначала привяжите компанию.</p>
          </div>

          {authError ? (
            <Notice tone="warning" title="Ошибка регистрации">
              {authError}
            </Notice>
          ) : null}

          {companyMessage ? (
            <Notice tone="success" title="Компания выбрана">
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
                    ? "Компания выбрана"
                    : "Привязать компанию"}
              </Button>
              {!isCompanyRegistered ? (
                <Button
                  disabled={companyForm.formState.isSubmitting}
                  onClick={continueWithoutCompany}
                  type="button"
                  variant="ghost"
                >
                  Продолжить без компании
                </Button>
              ) : null}
            </div>
          </form>

          <form className="stack-md" onSubmit={submitUser}>
            <Field label="Company ID (опционально)">
              <Input disabled value={createdCompany?.id ?? "Будет создана служебная компания"} />
            </Field>

            <Field label="Имя пользователя" error={registerUserForm.formState.errors.name?.message}>
              <Input placeholder="Менеджер продаж" {...registerUserForm.register("name")} />
            </Field>

            <Field label="Роль" error={registerUserForm.formState.errors.role?.message}>
              <Select {...registerUserForm.register("role")}>
                <option value="seller">Продавец</option>
                <option value="buyer">Покупатель</option>
              </Select>
            </Field>

            <Field label="Логин или email" error={registerUserForm.formState.errors.login?.message}>
              <Input placeholder="manager@north-sea.ru" {...registerUserForm.register("login")} />
            </Field>

            <Field label="Пароль" error={registerUserForm.formState.errors.password?.message}>
              <Input type="password" {...registerUserForm.register("password")} />
            </Field>

            <div className="stack-md">
              <label className="checkbox">
                <input
                  disabled={registerUserForm.formState.isSubmitting}
                  type="checkbox"
                  {...registerUserForm.register("acceptedTerms")}
                />
                <span>Я соглашаюсь с условиями пользования сервиса.</span>
              </label>
              {registerUserForm.formState.errors.acceptedTerms ? (
                <span className="field-error">{registerUserForm.formState.errors.acceptedTerms.message}</span>
              ) : null}
            </div>

            <div className="inline-actions">
              <Button
                disabled={
                  registerUserForm.formState.isSubmitting ||
                  !registerUserForm.watch("acceptedTerms")
                }
                type="submit"
              >
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
