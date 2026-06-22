// store にはログイン UI が無い (認証画面は mypage 側)。E2E は member サービスへ直接ログインして
// 得たトークンを Cookie に載せ、認証済み状態で store の注文確定を検証する。
export const MEMBER_API_URL = process.env.E2E_MEMBER_API_URL ?? "http://localhost:8004";

export const SESSION_COOKIE = "member_session";

export const MEMBER = {
  displayName: "E2E 太郎",
  email: "e2e@example.com",
  password: "e2e-password",
} as const;

async function login(): Promise<Response> {
  return fetch(`${MEMBER_API_URL}/sessions`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: MEMBER.email, password: MEMBER.password }),
  });
}

export async function ensureMember(): Promise<void> {
  // 複数回実行でも冪等にするため。
  const existing = await login();
  if (existing.ok) return;

  const created = await fetch(`${MEMBER_API_URL}/members`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(MEMBER),
  });
  if (created.status !== 201) throw new Error(`seed member failed: ${created.status}`);
}

export async function loginToken(): Promise<string> {
  const res = await login();
  if (res.status !== 201) throw new Error(`login failed: ${res.status}`);
  const session: { id: string } = await res.json();
  return session.id;
}
