import type { Meta, StoryObj } from "@storybook/react-vite";
import { PageLoading } from "ui/page-loading";

const meta: Meta = {
  title: "ui/PageLoading",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => <PageLoading />,
};
