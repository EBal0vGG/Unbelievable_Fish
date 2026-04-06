"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { createLot, createProduct, publishLot, publishProduct } from "@/shared/api/catalog-service";
import { isOwnedProduct } from "@/shared/lib/access";
import { toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";
import type { ServiceMeta } from "@/shared/types/domain";

const productSchema = z.object({
  fishId: z.string().min(1, "Выберите рыбу"),
  weight: z.coerce.number().positive("Введите вес"),
  unit: z.string().min(1, "Укажите единицу"),
  size: z.string().min(1, "Укажите размер"),
  processingType: z.string().min(1, "Укажите тип обработки"),
  publishProduct: z.boolean().default(true),
});

const lotSchema = z.object({
  productId: z.string().min(1, "Нужен productId"),
  photo: z.string().optional(),
  quantity: z.coerce.number().positive("Введите объем"),
  startPrice: z.coerce.number().int().positive("Введите стартовую цену"),
  auctionStartsAt: z.string().min(1, "Укажите старт торгов"),
  auctionDurationMinutes: z.coerce.number().int().positive("Введите длительность"),
  publishLot: z.boolean().default(true),
});

type ProductValues = z.infer<typeof productSchema>;
type LotValues = z.infer<typeof lotSchema>;

export function CreateLotForm() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [meta, setMeta] = useState<ServiceMeta | null>(null);
  const [productError, setProductError] = useState<string | null>(null);
  const [lotError, setLotError] = useState<string | null>(null);
  const [createdProductId, setCreatedProductId] = useState<string | null>(null);
  const [createdLotId, setCreatedLotId] = useState<string | null>(null);
  const [useManualProductId, setUseManualProductId] = useState(false);
  const fishQuery = useFishCatalogQuery(session);
  const productsQuery = useProductsQuery();
  const ownProducts = (productsQuery.data?.data ?? []).filter((product) => isOwnedProduct(product, session));

  const productForm = useForm<ProductValues>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      fishId: fishQuery.data?.data[0]?.id ?? "",
      weight: 20,
      unit: "kg",
      size: "2-4",
      processingType: "chilled",
      publishProduct: true,
    },
  });

  const lotForm = useForm<LotValues>({
    resolver: zodResolver(lotSchema),
    defaultValues: {
      productId: "",
      photo: "",
      quantity: 12,
      startPrice: 540000,
      auctionStartsAt: toDateTimeLocalValue(new Date(Date.now() + 60 * 60 * 1000)),
      auctionDurationMinutes: 180,
      publishLot: true,
    },
  });

  const productMutation = useMutation({
    mutationFn: async (values: ProductValues) => {
      setProductError(null);
      if (!session?.companyId || !session.userId) {
        throw new ApiError("Сначала сохраните пользовательский контекст", 400, "MISSING_SESSION");
      }
      const fish = fishQuery.data?.data.find((item) => item.id === values.fishId);
      const created = await createProduct(
        {
          fishId: values.fishId,
          fishName: fish?.name ?? values.fishId,
          weight: values.weight,
          unit: values.unit,
          size: values.size,
          processingType: values.processingType,
        },
        session,
      );

      if (values.publishProduct) {
        await publishProduct(created.data.id, session);
      }

      return created;
    },
    onSuccess: (result) => {
      setMeta(result.meta);
      setCreatedProductId(result.data.id);
      lotForm.setValue("productId", result.data.id);
      void queryClient.invalidateQueries({ queryKey: ["products"] });
    },
    onError: (error) => {
      setProductError(
        error instanceof ApiError
          ? error.message
          : "Не удалось создать продукт. Проверьте заполнение формы, сессию и доступность backend.",
      );
    },
  });

  const lotMutation = useMutation({
    mutationFn: async (values: LotValues) => {
      setLotError(null);
      if (!session?.companyId || !session.userId) {
        throw new ApiError("Сначала сохраните пользовательский контекст", 400, "MISSING_SESSION");
      }
      const product = productsQuery.data?.data.find((item) => item.id === values.productId);
      if (!product || !isOwnedProduct(product, session)) {
        throw new ApiError("lot can be created only from your own product", 403, "PRODUCT_ACCESS_DENIED");
      }
      if (values.publishLot && product?.status !== "PUBLISHED") {
        throw new ApiError(
          "product must be published before lot publish",
          400,
          "PUBLISHING_RULE_VIOLATION",
        );
      }
      const created = await createLot(
        {
          productId: values.productId,
          productLabel: product
            ? `${product.fishName} ${product.processingType} ${product.size} / ${product.weight} ${product.unit}`
            : values.productId,
          photo: values.photo,
          quantity: values.quantity,
          startPrice: values.startPrice,
          auctionStartsAt: new Date(values.auctionStartsAt).toISOString(),
          auctionDurationMinutes: values.auctionDurationMinutes,
        },
        session,
      );

      if (values.publishLot) {
        await publishLot(created.data.id, session);
      }

      return created;
    },
    onSuccess: (result) => {
      setMeta(result.meta);
      setCreatedLotId(result.data.id);
      void queryClient.invalidateQueries({ queryKey: ["lots"] });
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
    },
    onError: (error) => {
      setLotError(
        error instanceof ApiError
          ? error.message
          : "Не удалось создать лот. Проверьте productId, companyId и доступность backend.",
      );
    },
  });

  return (
    <div className="two-column-layout">
      <Card className="form-card">
        <div className="stack-lg">
          <div>
            <p className="eyebrow">Helper Product Flow</p>
            <h2>Быстрый продукт для лота</h2>
            <p className="muted">
              Лот в backend зависит от `product_id`, поэтому здесь есть вспомогательная форма
              создания продукта без отдельного top-level маршрута.
            </p>
          </div>

          {productError ? (
            <Notice tone="warning" title="Ошибка создания продукта">
              {productError}
            </Notice>
          ) : null}

          {createdProductId ? (
            <Notice tone="success" title="Продукт создан">
              `productId`: {createdProductId}
            </Notice>
          ) : null}

          <form className="stack-md" onSubmit={productForm.handleSubmit((values) => productMutation.mutate(values))}>
            <Field label="Рыба" error={productForm.formState.errors.fishId?.message}>
              <Select {...productForm.register("fishId")}>
                <option value="">Выберите рыбу</option>
                {(fishQuery.data?.data ?? []).map((fish) => (
                  <option key={fish.id} value={fish.id}>
                    {fish.name}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Вес" error={productForm.formState.errors.weight?.message}>
              <Input type="number" step="0.1" {...productForm.register("weight")} />
            </Field>
            <Field label="Единица" error={productForm.formState.errors.unit?.message}>
              <Select {...productForm.register("unit")}>
                <option value="kg">kg</option>
                <option value="g">g</option>
                <option value="ton">ton</option>
              </Select>
            </Field>
            <Field label="Размер" error={productForm.formState.errors.size?.message}>
              <Input placeholder="2-4" {...productForm.register("size")} />
            </Field>
            <Field label="Обработка" error={productForm.formState.errors.processingType?.message}>
              <Select {...productForm.register("processingType")}>
                <option value="chilled">chilled</option>
                <option value="frozen">frozen</option>
                <option value="live">live</option>
              </Select>
            </Field>
            <label className="checkbox">
              <input type="checkbox" {...productForm.register("publishProduct")} />
              <span>Сразу опубликовать продукт</span>
            </label>
            <Button disabled={productMutation.isPending || !session?.companyId || !session.userId} type="submit">
              {productMutation.isPending ? "Создаем..." : "Создать продукт"}
            </Button>
          </form>
        </div>
      </Card>

      <Card className="form-card">
        <div className="stack-lg">
          <div>
            <p className="eyebrow">Catalog Command</p>
            <h1>Создать лот</h1>
            <p className="muted">
              Реально подключены команды создания и публикации лота. Создание последующего auction
              read model зависит от integration runtime и пока не имеет стабильного query endpoint.
            </p>
          </div>

          {!session ? (
            <Notice tone="warning" title="Нужен вход">
              Для реального создания лота backend требует `companyId`. Сначала нажмите `Вход` или
              `Регистрация` и сохраните контекст пользователя.
            </Notice>
          ) : null}

          {meta?.note ? (
            <Notice tone={meta.source === "api" ? "success" : "warning"} title={`Источник: ${meta.source}`}>
              {meta.note}
            </Notice>
          ) : null}

          {lotError ? (
            <Notice tone="warning" title="Ошибка создания лота">
              {lotError}
            </Notice>
          ) : null}

          {createdLotId ? (
            <Notice tone="success" title="Лот создан">
              `lotId`: {createdLotId}
            </Notice>
          ) : null}

          <form className="stack-md" onSubmit={lotForm.handleSubmit((values) => lotMutation.mutate(values))}>
            <Field label="Продукт" error={lotForm.formState.errors.productId?.message}>
              <div className="stack-md">
                <div className="inline-actions inline-actions-start">
                  <Button
                    onClick={() => setUseManualProductId(false)}
                    size="sm"
                    type="button"
                    variant={useManualProductId ? "ghost" : "secondary"}
                  >
                    Выбрать из списка
                  </Button>
                  <Button
                    onClick={() => setUseManualProductId(true)}
                    size="sm"
                    type="button"
                    variant={useManualProductId ? "secondary" : "ghost"}
                  >
                    Ввести productId вручную
                  </Button>
                </div>

                {useManualProductId ? (
                  <Input placeholder="product-123" {...lotForm.register("productId")} />
                ) : (
                  <Select {...lotForm.register("productId")}>
                    <option value="">Выберите продукт</option>
                    {ownProducts.map((product) => (
                      <option key={product.id} value={product.id}>
                        {product.fishName} · {product.processingType} · {product.weight} {product.unit} · {product.status}
                      </option>
                    ))}
                  </Select>
                )}
              </div>
            </Field>
            <Field label="Фото URL">
              <Input placeholder="https://..." {...lotForm.register("photo")} />
            </Field>
            <Field label="Количество" error={lotForm.formState.errors.quantity?.message}>
              <Input type="number" step="0.1" {...lotForm.register("quantity")} />
            </Field>
            <Field label="Стартовая цена" error={lotForm.formState.errors.startPrice?.message}>
              <Input type="number" {...lotForm.register("startPrice")} />
            </Field>
            <Field label="Старт торгов" error={lotForm.formState.errors.auctionStartsAt?.message}>
              <Input type="datetime-local" {...lotForm.register("auctionStartsAt")} />
            </Field>
            <Field label="Длительность, минут" error={lotForm.formState.errors.auctionDurationMinutes?.message}>
              <Input type="number" {...lotForm.register("auctionDurationMinutes")} />
            </Field>
            <label className="checkbox">
              <input type="checkbox" {...lotForm.register("publishLot")} />
              <span>Сразу опубликовать лот</span>
            </label>
            <Button disabled={lotMutation.isPending || !session?.companyId || !session.userId} type="submit">
              {lotMutation.isPending ? "Сохраняем..." : "Создать лот"}
            </Button>
          </form>
        </div>
      </Card>
    </div>
  );
}
