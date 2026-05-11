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
import { isOwnedProduct, isSellerSession } from "@/shared/lib/access";
import { formatMoney, toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { EntityPhoto } from "@/shared/ui/entity-photo";
import { Field } from "@/shared/ui/field";
import { FormSection } from "@/shared/ui/form-section";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";

const productSchema = z.object({
  fishId: z.string().min(1, "Выберите рыбу"),
  weight: z.coerce.number().positive("Введите вес"),
  unit: z.string().min(1, "Укажите единицу"),
  size: z.string().min(1, "Укажите размер"),
  processingType: z.string().min(1, "Укажите тип обработки"),
  publishProduct: z.boolean().default(true),
});

const lotSchema = z.object({
  productId: z.string().min(1, "Выберите продукт"),
  photo: z.union([z.literal(""), z.string().trim().url("Укажите корректный URL")]).optional(),
  quantity: z.coerce.number().positive("Введите объем"),
  startPrice: z.coerce.number().int().positive("Введите стартовую цену"),
  minBidStep: z.coerce.number().int().positive("Введите минимальный шаг ставки"),
  auctionStartsAt: z.string().min(1, "Укажите старт торгов"),
  auctionDurationMinutes: z.coerce.number().int().positive("Введите длительность"),
  publishLot: z.boolean().default(true),
});

type ProductValues = z.infer<typeof productSchema>;
type LotValues = z.infer<typeof lotSchema>;

export function CreateLotForm() {
  const { session } = useAuth();
  const canManageLots = isSellerSession(session);
  const queryClient = useQueryClient();
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
      minBidStep: 1000,
      auctionStartsAt: toDateTimeLocalValue(new Date(Date.now() + 60 * 60 * 1000)),
      auctionDurationMinutes: 180,
      publishLot: true,
    },
  });

  const selectedProductId = lotForm.watch("productId");
  const photoUrl = lotForm.watch("photo");
  const startPrice = Number(lotForm.watch("startPrice") || 0);
  const selectedProduct = ownProducts.find((product) => product.id === selectedProductId);

  const productMutation = useMutation({
    mutationFn: async (values: ProductValues) => {
      setProductError(null);
      if (!session?.companyId || !session.userId) {
        throw new ApiError("Войдите в профиль, чтобы создать продукт", 400, "MISSING_SESSION");
      }
      if (!canManageLots) {
        throw new ApiError("Создание продукта доступно только продавцу", 403, "SELLER_ONLY");
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
      setCreatedProductId(result.data.id);
      lotForm.setValue("productId", result.data.id);
      void queryClient.invalidateQueries({ queryKey: ["products"] });
    },
    onError: (error) => {
      setProductError(
        error instanceof ApiError
          ? error.message
          : "Не удалось создать продукт. Проверьте заполнение формы и права доступа.",
      );
    },
  });

  const lotMutation = useMutation({
    mutationFn: async (values: LotValues) => {
      setLotError(null);
      if (!session?.companyId || !session.userId) {
        throw new ApiError("Войдите в профиль, чтобы создать лот", 400, "MISSING_SESSION");
      }
      if (!canManageLots) {
        throw new ApiError("Создание лота доступно только продавцу", 403, "SELLER_ONLY");
      }
      const product = productsQuery.data?.data.find((item) => item.id === values.productId);
      if (!product || !isOwnedProduct(product, session)) {
        throw new ApiError("Лот можно создать только из собственного продукта", 403, "PRODUCT_ACCESS_DENIED");
      }
      if (values.publishLot && product?.status !== "PUBLISHED") {
        throw new ApiError("Сначала опубликуйте продукт", 400, "PUBLISHING_RULE_VIOLATION");
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
          minBidStep: values.minBidStep,
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
      setCreatedLotId(result.data.id);
      void queryClient.invalidateQueries({ queryKey: ["lots"] });
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
    },
    onError: (error) => {
      setLotError(
        error instanceof ApiError
          ? error.message
          : "Не удалось создать лот. Проверьте данные формы и выбранный продукт.",
      );
    },
  });

  return (
    <div className="page-stack">
      <section className="workspace-page-header">
        <div>
          <p className="eyebrow">Новая поставка</p>
          <h1>Создать лот</h1>
          <p className="hero-copy">Соберите продукт, добавьте фото партии и сразу подготовьте лот к торгам.</p>
        </div>
      </section>

      {!session ? (
        <Notice tone="warning" title="Нужен вход">
          Войдите в профиль, чтобы создавать продукты и лоты.
        </Notice>
      ) : !canManageLots ? (
        <Notice tone="warning" title="Доступно продавцам">
          Создание продуктов и лотов доступно только компании-поставщику.
        </Notice>
      ) : session && (!session.companyId || !session.userId) ? (
        <Notice tone="warning" title="Нет привязки к компании">
          В профиле пустой company_id (старый аккаунт или сбой). Выйдите и зарегистрируйтесь снова — для новых пользователей без своей компании identity создаёт персональную организацию автоматически.
        </Notice>
      ) : null}

      <div className="supply-builder">
        <aside className="supply-preview">
          <EntityPhoto src={photoUrl} alt={selectedProduct?.fishName ?? "Фото лота"} className="detail-photo" />
          <div className="preview-total">
            <span>Стартовая цена лота</span>
            <strong>{formatMoney(startPrice)}</strong>
            <p>Покупатель делает ставку за весь лот.</p>
          </div>
          <div className="supply-panel stack-md compact-card">
            <div>
              <p className="eyebrow">Выбранный продукт</p>
              <h2>{selectedProduct?.fishName ?? "Продукт не выбран"}</h2>
              <p className="muted">
                {selectedProduct
                  ? `${selectedProduct.processingType} · ${selectedProduct.size} · ${selectedProduct.weight} ${selectedProduct.unit}`
                  : "Выберите продукт из списка или создайте новый слева."}
              </p>
            </div>
          </div>
        </aside>

        <div className="stack-lg">
          <div className="wizard-steps" aria-label="Шаги создания лота">
            <span className="wizard-step-active">1. Продукт</span>
            <span>2. Партия</span>
            <span>3. Торги</span>
            <span>4. Публикация</span>
          </div>
          <FormSection
            title="1. Продукт"
            description="Создайте позицию каталога и используйте ее в лоте без перехода на другую страницу."
          >

            {productError ? (
              <Notice tone="warning" title="Не удалось создать продукт">
                {productError}
              </Notice>
            ) : null}

            {createdProductId ? (
              <Notice tone="success" title="Продукт готов">
                Продукт выбран для нового лота.
              </Notice>
            ) : null}

            <form className="form-grid-2" onSubmit={productForm.handleSubmit((values) => productMutation.mutate(values))}>
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
              <Field label="Обработка" error={productForm.formState.errors.processingType?.message}>
                <Select {...productForm.register("processingType")}>
                  <option value="chilled">Охлажденная</option>
                  <option value="frozen">Замороженная</option>
                  <option value="live">Живая</option>
                </Select>
              </Field>
              <Field label="Вес единицы" error={productForm.formState.errors.weight?.message}>
                <Input type="number" step="0.1" {...productForm.register("weight")} />
              </Field>
              <Field label="Единица" error={productForm.formState.errors.unit?.message}>
                <Select {...productForm.register("unit")}>
                  <option value="kg">кг</option>
                  <option value="g">г</option>
                  <option value="ton">тонна</option>
                </Select>
              </Field>
              <Field label="Размер" error={productForm.formState.errors.size?.message}>
                <Input placeholder="2-4" {...productForm.register("size")} />
              </Field>
              <label className="checkbox">
                <input type="checkbox" {...productForm.register("publishProduct")} />
                <span>Опубликовать продукт после создания</span>
              </label>
              <div className="inline-actions form-grid-full">
                <Button
                  disabled={productMutation.isPending || !session?.companyId || !session.userId || !canManageLots}
                  type="submit"
                >
                  {productMutation.isPending ? "Создаем..." : "Создать продукт"}
                </Button>
              </div>
            </form>
          </FormSection>

          <form className="stack-lg" onSubmit={lotForm.handleSubmit((values) => lotMutation.mutate(values))}>
            <FormSection
              title="2. Партия"
              description="Выберите продукт, объем и фото партии. Фото будет показано в карточке лота и на странице аукциона."
            >

            {lotError ? (
              <Notice tone="warning" title="Не удалось создать лот">
                {lotError}
              </Notice>
            ) : null}

            {createdLotId ? (
              <Notice tone="success" title="Лот создан">
                Лот готов к торгам.
              </Notice>
            ) : null}

            <div className="form-grid-2">
              <Field className="form-grid-full" label="Продукт" error={lotForm.formState.errors.productId?.message}>
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
                      Ввести номер вручную
                    </Button>
                  </div>

                  {useManualProductId ? (
                    <Input placeholder="Номер продукта" {...lotForm.register("productId")} />
                  ) : (
                    <Select {...lotForm.register("productId")}>
                      <option value="">Выберите продукт</option>
                      {ownProducts.map((product) => (
                        <option key={product.id} value={product.id}>
                          {product.fishName} · {product.processingType} · {product.weight} {product.unit}
                        </option>
                      ))}
                    </Select>
                  )}
                </div>
              </Field>
              <Field className="form-grid-full" label="Фото партии" error={lotForm.formState.errors.photo?.message}>
                <Input placeholder="https://example.com/fish-lot.jpg" {...lotForm.register("photo")} />
              </Field>
              <Field label="Объем партии" error={lotForm.formState.errors.quantity?.message}>
                <Input type="number" step="0.1" {...lotForm.register("quantity")} />
              </Field>
              <Field
                label="Стартовая цена лота"
                error={lotForm.formState.errors.startPrice?.message}
                hint="Покупатель делает ставку за весь лот."
              >
                <Input type="number" {...lotForm.register("startPrice")} />
              </Field>
            </div>
            </FormSection>

            <FormSection title="3. Параметры торгов" description="Настройте минимальный шаг и окно приема ставок.">
            <div className="form-grid-2">
              <Field label="Мин. шаг ставки" error={lotForm.formState.errors.minBidStep?.message}>
                <Input type="number" {...lotForm.register("minBidStep")} />
              </Field>
              <Field label="Старт торгов" error={lotForm.formState.errors.auctionStartsAt?.message}>
                <Input type="datetime-local" {...lotForm.register("auctionStartsAt")} />
              </Field>
              <Field label="Длительность, минут" error={lotForm.formState.errors.auctionDurationMinutes?.message}>
                <Input type="number" {...lotForm.register("auctionDurationMinutes")} />
              </Field>
            </div>
            </FormSection>

            <FormSection title="4. Публикация" description="Публикация запускает дальнейшую integration-цепочку торгов.">
            <div className="form-grid-2">
              <label className="checkbox form-grid-full">
                <input type="checkbox" {...lotForm.register("publishLot")} />
                <span>Опубликовать лот после создания</span>
              </label>
              <div className="inline-actions form-grid-full">
                <Button
                  disabled={lotMutation.isPending || !session?.companyId || !session.userId || !canManageLots}
                  type="submit"
                >
                  {lotMutation.isPending ? "Сохраняем..." : "Создать лот"}
                </Button>
              </div>
            </div>
            </FormSection>
          </form>
        </div>
      </div>
    </div>
  );
}
