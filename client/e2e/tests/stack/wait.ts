// docker compose の --wait は healthcheck を持つサービスしか待てない。frontend と各 API は
// healthcheck を持たないため、HTTP 到達で readiness を判定する。
export async function waitForOk(url: string, timeoutMs = 120_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
      lastError = new Error(`GET ${url} returned ${res.status}`);
    } catch (e) {
      lastError = e;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`not ready at ${url}: ${String(lastError)}`);
}
