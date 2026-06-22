import type { Preview } from "@storybook/react-vite";
import "ui/styles.css";

const preview: Preview = {
  parameters: {
    layout: "centered",
    controls: { disable: true },
  },
  decorators: [
    (Story) => (
      <div className="bg-background text-foreground p-6">
        <Story />
      </div>
    ),
  ],
};

export default preview;
