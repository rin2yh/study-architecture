import { LoginForm } from "@/features/auth";

export function LoginPage({ actionData }: { actionData?: { error: string } }) {
  return <LoginForm error={actionData?.error} />;
}
