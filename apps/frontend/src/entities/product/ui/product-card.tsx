import { Badge } from "@/shared/ui/badge";
import { Card } from "@/shared/ui/card";
import { shortId } from "@/shared/lib/format";
import type { ProductRecord } from "@/shared/types/domain";

function productTone(status: ProductRecord["status"]) {
  switch (status) {
    case "PUBLISHED":
      return "success";
    default:
      return "warning";
  }
}

export function ProductCard({ product }: { product: ProductRecord }) {
  return (
    <Card className="entity-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Продукт</p>
          <h3>{product.fishName}</h3>
        </div>
        <Badge tone={productTone(product.status)}>{product.status}</Badge>
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
          <strong>{product.ownerCompanyId}</strong>
        </div>
      </div>
      <p className="entity-card-meta">Позиция #{shortId(product.id)}</p>
    </Card>
  );
}
