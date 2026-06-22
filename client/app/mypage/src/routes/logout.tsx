import { deleteSession } from "api/member";
import { redirect } from "react-router";
import type { Route } from "./+types/logout";
import { clearSessionCookie, readSessionToken } from "@/entities/session";

export async function action({ request }: Route.ActionArgs) {
  const token = readSessionToken(request);
  if (token) {
    // ADR-[[202606211100]]
    try {
      await deleteSession(token);
    } catch (e) {
      console.warn("logout: server session の破棄に失敗", e);
    }
  }
  return redirect("/login", { headers: { "Set-Cookie": clearSessionCookie() } });
}

export async function loader() {
  throw redirect("/login");
}
