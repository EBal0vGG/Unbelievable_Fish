import Link from "next/link";
import { buttonStyles } from "@/shared/ui/button";

export default function CreateAuctionPage() {
  return (
    <section className="form-card">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Торги</p>
          <h1>Ручное создание аукциона отключено</h1>
          <p className="muted">
            Аукцион создаётся автоматически после публикации лота в Catalog через integration-цепочку событий.
          </p>
        </div>
        <div className="inline-actions">
          <Link className={buttonStyles({ variant: "secondary", size: "md" })} href="/create/lot">
            Перейти к созданию лота
          </Link>
          <Link className={buttonStyles({ variant: "ghost", size: "md" })} href="/auctions">
            К списку аукционов
          </Link>
        </div>
      </div>
    </section>
  );
}
