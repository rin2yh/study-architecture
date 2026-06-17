import { defineConfig } from "vite";
import { devtools } from "@tanstack/devtools-vite";

import { tanstackStart } from "@tanstack/react-start/plugin/vite";

import viteReact from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// nitro/vite plugin は production build に Vite-dev 用 SSR fallback
// (fetch(req,{viteEnv:"ssr"})) を残しており、Docker 起動で自己 fetch がデッドロックする
// (doc/known-issues.md 参照)。tanstackStart() の SSR ハンドラ (dist/server/server.js)
// を自前の薄い Node http サーバから呼ぶ構成で SSR を保つ。
const config = defineConfig({
  resolve: { tsconfigPaths: true },
  plugins: [devtools(), tailwindcss(), tanstackStart(), viteReact()],
});

export default config;
