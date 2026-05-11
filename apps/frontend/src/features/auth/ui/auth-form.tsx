"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { registerCompany, registerUser, resendVerification } from "@/shared/api/identity-service";
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
  role: z.enum(["seller", "buyer", "buyer_seller"]),
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
  const [registrationEmail, setRegistrationEmail] = useState<string | null>(null);
  const [unverifiedLogin, setUnverifiedLogin] = useState<string | null>(null);
  const [resendMessage, setResendMessage] = useState<string | null>(null);
  const [isResending, setIsResending] = useState(false);
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
    setResendMessage(null);
    setUnverifiedLogin(null);
    try {
      await login(values.login, values.password);
      router.push(nextPath);
    } catch (error) {
      if (error instanceof ApiError && error.code === "EMAIL_NOT_VERIFIED") {
        setUnverifiedLogin(values.login);
        setAuthError("Email не подтвержден. Проверьте почту или отправьте письмо повторно.");
        return;
      }
      setAuthError(error instanceof ApiError ? mapIdentityError(error) : "Не удалось войти в систему.");
    }
  });

  const submitCompany = companyForm.handleSubmit(async (values) => {
    setAuthError(null);
    try {
      const company = await registerCompany(values);
      setCreatedCompany({ id: company.id, name: company.name });
      setCompanyMessage(`Компания ${company.name} выбрана. Теперь создайте пользователя.`);
    } catch (error) {
      setAuthError(error instanceof ApiError ? mapIdentityError(error) : "Не удалось зарегистрировать компанию.");
    }
  });

  const continueWithoutCompany = () => {
    setAuthError(null);
    setCreatedCompany(null);
    setCompanyMessage(
      "Компания не заполнена — identity создаст персональную организацию для вашего аккаунта (для торгов и каталога).",
    );
  };

  const submitUser = registerUserForm.handleSubmit(async (values) => {
    setAuthError(null);
    setResendMessage(null);
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
      setRegistrationEmail(values.login);
    } catch (error) {
      if (error instanceof ApiError && error.code === "EMAIL_SEND_FAILED") {
        setRegistrationEmail(values.login);
        setAuthError("Аккаунт создан, но письмо подтверждения не удалось отправить. Попробуйте отправить письмо повторно.");
        return;
      }
      setAuthError(error instanceof ApiError ? mapIdentityError(error) : "Не удалось завершить регистрацию.");
    }
  });

  const resend = async (loginValue: string) => {
    setIsResending(true);
    setResendMessage(null);
    setAuthError(null);
    try {
      const result = await resendVerification(loginValue);
      setResendMessage(
        result.already_verified
          ? "Email уже подтвержден. Теперь можно войти."
          : "Письмо подтверждения отправлено повторно.",
      );
    } catch (error) {
      setAuthError(error instanceof ApiError ? mapIdentityError(error) : "Не удалось отправить письмо повторно.");
    } finally {
      setIsResending(false);
    }
  };

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
            {resendMessage ? (
              <Notice tone="success" title="Письмо подтверждения">
                {resendMessage}
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
                {unverifiedLogin ? (
                  <Button disabled={isResending} onClick={() => resend(unverifiedLogin)} type="button" variant="secondary">
                    {isResending ? "Отправляем..." : "Отправить письмо повторно"}
                  </Button>
                ) : null}
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

  if (registrationEmail) {
    return (
      <div className="auth-shell">
        <Card className="auth-card">
          <div className="stack-lg">
            <div>
              <h1>Проверьте почту</h1>
              <p className="muted">Регистрация почти завершена. Мы отправили письмо для подтверждения email.</p>
            </div>

            {authError ? (
              <Notice tone="warning" title="Не удалось отправить письмо">
                {authError}
              </Notice>
            ) : null}
            {resendMessage ? (
              <Notice tone="success" title="Письмо подтверждения">
                {resendMessage}
              </Notice>
            ) : null}

            <div className="inline-actions">
              <Button disabled={isResending} onClick={() => resend(registrationEmail)} type="button">
                {isResending ? "Отправляем..." : "Отправить письмо повторно"}
              </Button>
              <Link className={buttonStyles({ variant: "ghost" })} href="/login">
                Перейти ко входу
              </Link>
            </div>
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
              <Input
                disabled
                value={createdCompany?.id ?? "Персональная компания создаётся на сервере, если не привязали свою"}
              />
            </Field>

            <Field label="Имя пользователя" error={registerUserForm.formState.errors.name?.message}>
              <Input placeholder="Менеджер продаж" {...registerUserForm.register("name")} />
            </Field>

            <Field label="Роль" error={registerUserForm.formState.errors.role?.message}>
              <Select {...registerUserForm.register("role")}>
                <option value="seller">Продавец</option>
                <option value="buyer">Покупатель</option>
                <option value="buyer_seller">Без роли</option>
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
                {registerUserForm.formState.isSubmitting ? "Создаем..." : "Создать пользователя"}
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

function mapIdentityError(error: ApiError): string {
  switch (error.code) {
    case "EMAIL_NOT_VERIFIED":
      return "Email не подтвержден. Проверьте почту или отправьте письмо повторно.";
    case "VERIFICATION_COOLDOWN":
      return "Письмо уже отправлено. Повторная отправка будет доступна через несколько минут.";
    case "EMAIL_SEND_FAILED":
      return "Сервис отправки писем временно недоступен. Попробуйте позже.";
    case "UPSTREAM_TIMEOUT":
    case "UPSTREAM_UNAVAILABLE":
      return "Сервис авторизации временно недоступен. Попробуйте еще раз.";
    case "INVALID_CREDENTIALS":
      return "Неверный логин или пароль.";
    case "LOGIN_ALREADY_USED":
      return "Пользователь с таким email уже зарегистрирован.";
    default:
      if (error.status === 0) {
        return "Сервис авторизации временно недоступен. Попробуйте еще раз.";
      }
      return error.message;
  }
}
