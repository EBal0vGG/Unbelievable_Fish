"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { createFish } from "@/shared/api/catalog-service";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Textarea } from "@/shared/ui/textarea";
import type { ServiceMeta } from "@/shared/types/domain";

const schema = z.object({
  name: z.string().min(2, "Название рыбы обязательно"),
  description: z.string().min(4, "Добавьте короткое описание"),
});

type Values = z.infer<typeof schema>;

export function CreateFishForm() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [meta, setMeta] = useState<ServiceMeta | null>(null);

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: "",
      description: "",
    },
  });

  const mutation = useMutation({
    mutationFn: (values: Values) => createFish(values, session),
    onSuccess: (result) => {
      setMeta(result.meta);
      form.reset();
      void queryClient.invalidateQueries({ queryKey: ["fish-catalog"] });
    },
  });

  return (
    <Card className="form-card">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Catalog Command</p>
          <h1>Создать рыбу</h1>
          <p className="muted">
            Форма подключена к `POST /fish`, а при недоступном route аккуратно падает в локальный
            placeholder.
          </p>
        </div>

        {meta?.note ? (
          <Notice tone={meta.source === "api" ? "success" : "warning"} title={`Источник: ${meta.source}`}>
            {meta.note}
          </Notice>
        ) : null}

        <form className="stack-md" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <Field label="Название" error={form.formState.errors.name?.message}>
            <Input placeholder="Нерка камчатская" {...form.register("name")} />
          </Field>
          <Field label="Описание" error={form.formState.errors.description?.message}>
            <Textarea rows={4} placeholder="Оптовая позиция для каталога..." {...form.register("description")} />
          </Field>
          <div className="inline-actions">
            <Button disabled={mutation.isPending} type="submit">
              {mutation.isPending ? "Сохраняем..." : "Создать рыбу"}
            </Button>
          </div>
        </form>
      </div>
    </Card>
  );
}
