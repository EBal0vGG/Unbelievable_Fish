"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { isAdminSession } from "@/shared/lib/access";
import { ApiError } from "@/shared/api/http-client";
import { createFish } from "@/shared/api/catalog-service";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Textarea } from "@/shared/ui/textarea";

const schema = z.object({
  name: z.string().min(2, "Название рыбы обязательно"),
  description: z.string().min(4, "Добавьте короткое описание"),
});

type Values = z.infer<typeof schema>;

export function CreateFishForm() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const canManageFish = isAdminSession(session);

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: "",
      description: "",
    },
  });

  const mutation = useMutation({
    mutationFn: (values: Values) => {
      setError(null);
      return createFish(values, session);
    },
    onSuccess: () => {
      form.reset();
      void queryClient.invalidateQueries({ queryKey: ["fish-catalog"] });
    },
    onError: (error) => {
      setError(error instanceof ApiError ? error.message : "Не удалось создать рыбу.");
    },
  });

  return (
    <Card className="form-card">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Каталог</p>
          <h1>Создать рыбу</h1>
          <p className="muted">Добавьте новую позицию в каталог.</p>
        </div>

        {!canManageFish ? (
          <Notice tone="warning" title="Доступ ограничен">
            Создание рыбы доступно только администраторам.
          </Notice>
        ) : null}

        {error ? (
          <Notice tone="warning" title="Ошибка создания рыбы">
            {error}
          </Notice>
        ) : null}

        <form className="stack-md" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <Field label="Название" error={form.formState.errors.name?.message}>
            <Input disabled={!canManageFish} placeholder="Нерка камчатская" {...form.register("name")} />
          </Field>
          <Field label="Описание" error={form.formState.errors.description?.message}>
            <Textarea disabled={!canManageFish} rows={4} placeholder="Оптовая позиция для каталога..." {...form.register("description")} />
          </Field>
          <div className="inline-actions">
            <Button disabled={mutation.isPending || !canManageFish} type="submit">
              {mutation.isPending ? "Сохраняем..." : "Создать рыбу"}
            </Button>
          </div>
        </form>
      </div>
    </Card>
  );
}
