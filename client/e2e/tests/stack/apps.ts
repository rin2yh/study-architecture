export const apps = ["store", "backoffice"] as const;
export type App = (typeof apps)[number];

export const baseURLs: Record<App, string> = {
  store: process.env.E2E_BASE_URL ?? "http://localhost:5173",
  backoffice: process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175",
};
