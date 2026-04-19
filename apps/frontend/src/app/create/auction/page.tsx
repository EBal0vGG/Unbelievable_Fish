import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { CreateAuctionForm } from "@/features/auction/ui/create-auction-form";

export default function CreateAuctionPage() {
  return (
    <AuthGuard roles={["seller"]}>
      <CreateAuctionForm />
    </AuthGuard>
  );
}
