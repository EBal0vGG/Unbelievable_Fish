import { Card } from "@/shared/ui/card";
import { displayCompany } from "@/shared/lib/display";
import { shortId } from "@/shared/lib/format";
import { productStatusLabels } from "@/shared/lib/labels";
import { StatusBadge } from "@/shared/ui/status-badge";
import type { ProductRecord } from "@/shared/types/domain";

export function ProductCard({ product }: { product: ProductRecord }) {
  return (
    <Card className="entity-card product-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Продукт</p>
          <h3>{product.fishName}</h3>
          <p className="entity-card-meta">Позиция #{shortId(product.id)}</p>
        </div>
        <StatusBadge status={product.status} label={productStatusLabels[product.status]} />
      </div>
      <div className="metric-grid">
        <div>
          <span>Обработка</span>
          <strong>{product.processingType}</strong>
        </div>
        <div>
          <span>Размер</span>
          <strong>{product.size}</strong>
        </div>
        <div>
          <span>Вес</span>
          <strong>
            {product.weight} {product.unit}
          </strong>
        </div>
        <div>
          <span>Компания</span>
          <strong title={product.ownerCompanyId}>{displayCompany(product.ownerCompanyId)}</strong>
        </div>
      </div>
      <div className="entity-card-footer">
        <span>Категория</span>
        <strong>{product.processingType}</strong>
      </div>
    </Card>
  );
}
