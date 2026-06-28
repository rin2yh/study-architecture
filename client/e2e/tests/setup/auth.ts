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

export const MEMBER_ADDRESS = {
  recipient: "E2E 太郎",
  postalCode: "1500001",
  prefecture: "東京都",
  city: "渋谷区",
  line1: "神宮前1-2-3",
} as const;

async function memberIdOf(loginRes: Response): Promise<number> {
  const { memberId }: { memberId: number } = await loginRes.json();
  return memberId;
}

export async function ensureMember(): Promise<void> {
  // 複数回実行でも冪等にするため。
  const existing = await login();
  if (existing.ok) {
    await ensureAddress(await memberIdOf(existing));
    return;
  }

  const created = await fetch(`${MEMBER_API_URL}/members`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(MEMBER),
  });
  if (created.status !== 201) throw new Error(`seed member failed: ${created.status}`);
  const { id }: { id: number } = await created.json();
  await ensureAddress(id);
}

// checkout は配送先 (住所帳の1件) を要求する (ADR-[[202606261704]])。住所が無いと確定できないため
// 会員に最低1件用意する。再実行で重複しないよう既存があればスキップする。
async function ensureAddress(memberId: number): Promise<void> {
  const list = await fetch(`${MEMBER_API_URL}/members/${memberId}/addresses`);
  if (!list.ok) throw new Error(`list addresses failed: ${list.status}`);
  const existing: unknown[] = await list.json();
  if (existing.length > 0) return;

  const created = await fetch(`${MEMBER_API_URL}/members/${memberId}/addresses`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(MEMBER_ADDRESS),
  });
  if (created.status !== 201) throw new Error(`seed address failed: ${created.status}`);
}
