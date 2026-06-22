import type { StorybookConfig } from "@storybook/react-vite";

const config: StorybookConfig = {
  framework: "@storybook/react-vite",
  stories: ["../src/**/*.stories.@(ts|tsx)"],
  addons: [],
  core: {
    // 既定だと store の vite.config.ts (reactRouter プラグイン) を読み込んで衝突するため。
    builder: {
      name: "@storybook/builder-vite",
      options: { viteConfigPath: ".storybook/vite.config.ts" },
    },
  },
};

export default config;
