import { getSession, GetSessionResponse } from "api/member";

export const SESSION_COOKIE = "member_session";

// ADR-[[202606211100]]
const MAX_AGE_SEC = 7 * 24 * 60 * 60;

export function readSessionToken(request: Request): string | null {
  const header = request.headers.get("Cookie");
  if (!header) return null;
  for (const part of header.split(";")) {
    const eq = part.indexOf("=");
    if (eq === -1) continue;
    if (part.slice(0, eq).trim() === SESSION_COOKIE) {
      return decodeURIComponent(part.slice(eq + 1).trim());
    }
  }
  return null;
}

// ADR-[[202606211100]]
export function sessionCookie(token: string): string {
  return `${SESSION_COOKIE}=${encodeURIComponent(token)}; Path=/; HttpOnly; SameSite=Lax; Max-Age=${MAX_AGE_SEC}`;
}

export function clearSessionCookie(): string {
  return `${SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`;
}

// ADR-[[202606211100]]
export async function currentMemberId(request: Request): Promise<number | null> {
  const token = readSessionToken(request);
  if (!token) return null;
  try {
    const { data } = await getSession(token);
    return GetSessionResponse.parse(data).memberId;
  } catch {
    return null;
  }
}
