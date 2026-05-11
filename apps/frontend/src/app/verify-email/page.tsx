import { VerifyEmailCard } from "@/features/auth/ui/verify-email-card";

export default async function VerifyEmailPage({
  searchParams,
}: {
  searchParams?: Promise<{ token?: string }>;
}) {
  const params = (await searchParams) ?? {};
  return <VerifyEmailCard token={params.token} />;
}
