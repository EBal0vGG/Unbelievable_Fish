import { AuthForm } from "@/features/auth/ui/auth-form";

export default async function RegisterPage({
  searchParams,
}: {
  searchParams?: Promise<{ next?: string }>;
}) {
  const params = (await searchParams) ?? {};
  return <AuthForm mode="register" nextPath={params.next || "/"} />;
}
