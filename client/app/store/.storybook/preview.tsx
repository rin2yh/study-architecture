import type { Preview } from "@storybook/react-vite";
import "ui/styles.css";

const preview: Preview = {
  parameters: {
    layout: "fullscreen",
    controls: { disable: true },
  },
  decorators: [
    (Story) => (
      <div className="bg-background text-foreground min-h-screen">
        <Story />
      </div>
    ),
  ],
};

export default preview;
