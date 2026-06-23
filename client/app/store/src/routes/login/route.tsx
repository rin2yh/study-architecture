import { createSession, CreateSessionResponse } from "api/member";
import { redirect } from "react-router";
import type { Route } from "./+types/route";
import { redirectIfAuthenticated, sessionCookie } from "@/entities/session";
import { LoginForm } from "@/features/auth";

export async function loader({ request }: Route.LoaderArgs) {
  await redirectIfAuthenticated(request);
  return null;
}

export async function action({ request }: Route.ActionArgs) {
  const form = await request.formData();
  const email = String(form.get("email") ?? "");
  const password = String(form.get("password") ?? "");
  try {
    const { data } = await createSession({ email, password });
    const { id } = CreateSessionResponse.parse(data);
    return redirect("/", { headers: { "Set-Cookie": sessionCookie(id) } });
  } catch {
    // ADR-[[202606211100]]
    return { error: "メールアドレスまたはパスワードが違います" };
  }
}

export default function Login({ actionData }: Route.ComponentProps) {
  return <LoginForm error={actionData?.error} />;
}
