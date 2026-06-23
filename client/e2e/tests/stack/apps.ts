export const apps = ["store", "backoffice"] as const;
export type App = (typeof apps)[number];

interface AppConfig {
  profile: string;
  baseURL: string;
  // ログインフローがある app だけ member シードが要る。閲覧のみなら false。
  needsMember: boolean;
}

export const appConfig: Record<App, AppConfig> = {
  store: {
    profile: "external",
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:5173",
    needsMember: true,
  },
  backoffice: {
    profile: "internal",
    baseURL: process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175",
    needsMember: false,
  },
};

// setup / teardown project は app ごとに同じ spec を共有し、project 名 (`<app>-setup` 等) から
// 対象 app を引く。
export function appFromProjectName(name: string): App {
  const candidate = name.replace(/-(setup|teardown)$/, "");
  const app = apps.find((a) => a === candidate);
  if (!app) throw new Error(`unknown app project: ${name}`);
  return app;
}
