import tailwindcss from "@tailwindcss/vite";
import tsconfigPaths from "vite-tsconfig-paths";
import { defineConfig } from "vite";

// react プラグインは @storybook/react-vite が自動注入するため (二重登録を避ける)。
export default defineConfig({
  plugins: [tsconfigPaths(), tailwindcss()],
});
