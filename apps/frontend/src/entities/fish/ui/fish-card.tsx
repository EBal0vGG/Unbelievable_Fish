import { Card } from "@/shared/ui/card";
import type { FishRecord } from "@/shared/types/domain";

export function FishCard({ fish }: { fish: FishRecord }) {
  return (
    <Card className="entity-card fish-card catalog-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Рыба</p>
          <h3>{fish.name}</h3>
        </div>
        <span className="category-chip">Каталог</span>
      </div>
      <p className="muted">{fish.description}</p>
      <div className="entity-card-footer">
        <span>Раздел</span>
        <strong>Справочник платформы</strong>
      </div>
    </Card>
  );
}
