import { deleteSession } from "api/member";
import { redirect } from "react-router";
import type { Route } from "./+types/logout";
import { clearSessionCookie, readSessionToken } from "../session";

export async function action({ request }: Route.ActionArgs) {
  const token = readSessionToken(request);
  if (token) {
    // サーバ側セッションの破棄に失敗しても、Cookie は必ず消してログアウトを成立させる。
    try {
      await deleteSession(token);
    } catch {
      /* noop */
    }
  }
  return redirect("/login", { headers: { "Set-Cookie": clearSessionCookie() } });
}

export async function loader() {
  throw redirect("/login");
}
