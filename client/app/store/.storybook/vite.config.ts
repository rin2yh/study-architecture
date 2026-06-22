import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// react プラグインは @storybook/react-vite が自動注入するため、ここでは Tailwind だけ足す。
// theme.css の @source と合わせて ui コンポーネントの class を生成させる。
export default defineConfig({
  plugins: [tailwindcss()],
});
