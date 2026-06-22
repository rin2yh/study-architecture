// テストは store の /login 画面からログインする。member は事前に存在している必要があるため、
// member サービスへ直接シードする。
export const MEMBER_API_URL = process.env.E2E_MEMBER_API_URL ?? "http://localhost:8004";

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
