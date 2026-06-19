import { createSession, CreateSessionResponse } from "api/member";
import { Form, redirect } from "react-router";
import type { Route } from "./+types/login";
import { currentMemberId, sessionCookie } from "../session";

export async function loader({ request }: Route.LoaderArgs) {
  if ((await currentMemberId(request)) !== null) throw redirect("/");
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
    // 401 と他の失敗を UI で区別しない (member 側で user enumeration を避けているため)。
    return { error: "メールアドレスまたはパスワードが違います" };
  }
}

export default function Login({ actionData }: Route.ComponentProps) {
  return (
    <div className="mx-auto max-w-sm p-8">
      <h1 className="text-3xl font-bold">ログイン</h1>
      <Form method="post" className="mt-6 flex flex-col gap-4">
        <label className="flex flex-col gap-1 text-sm">
          メールアドレス
          <input
            type="email"
            name="email"
            required
            autoComplete="email"
            className="rounded border border-gray-300 px-3 py-2"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          パスワード
          <input
            type="password"
            name="password"
            required
            autoComplete="current-password"
            className="rounded border border-gray-300 px-3 py-2"
          />
        </label>
        {actionData?.error && (
          <p role="alert" className="text-sm text-red-600">
            {actionData.error}
          </p>
        )}
        <button
          type="submit"
          className="rounded bg-gray-900 px-3 py-2 text-white hover:bg-gray-700"
        >
          ログイン
        </button>
      </Form>
    </div>
  );
}
