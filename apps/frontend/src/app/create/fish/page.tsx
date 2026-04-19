import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { CreateFishForm } from "@/features/fish/ui/create-fish-form";

export default function CreateFishPage() {
  return (
    <AuthGuard roles={["admin"]}>
      <CreateFishForm />
    </AuthGuard>
  );
}
