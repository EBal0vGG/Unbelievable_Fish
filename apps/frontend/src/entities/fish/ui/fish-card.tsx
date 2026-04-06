import { Card } from "@/shared/ui/card";
import { Badge } from "@/shared/ui/badge";
import { shortId } from "@/shared/lib/format";
import type { FishRecord } from "@/shared/types/domain";

export function FishCard({ fish }: { fish: FishRecord }) {
  return (
    <Card className="entity-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Рыба</p>
          <h3>{fish.name}</h3>
        </div>
        <Badge tone={fish.source === "api" ? "success" : "warning"}>{fish.source}</Badge>
      </div>
      <p className="muted">{fish.description}</p>
      <p className="entity-card-meta">{shortId(fish.id)}</p>
    </Card>
  );
}
