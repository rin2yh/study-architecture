import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// react プラグインは @storybook/react-vite が自動注入するため (二重登録を避ける)。
export default defineConfig({
  plugins: [tailwindcss()],
  resolve: {
    // store の tsconfig paths (@/*) と揃えるため。
    alias: { "@": fileURLToPath(new URL("../src", import.meta.url)) },
  },
});
