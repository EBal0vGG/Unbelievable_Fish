import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { CreateLotForm } from "@/features/lot/ui/create-lot-form";

export default function CreateLotPage() {
  return (
    <AuthGuard roles={["seller"]}>
      <CreateLotForm />
    </AuthGuard>
  );
}
