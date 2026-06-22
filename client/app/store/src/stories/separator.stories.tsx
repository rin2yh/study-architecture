import type { Meta, StoryObj } from "@storybook/react-vite";
import { Separator } from "ui/separator";

const meta: Meta = {
  title: "ui/Separator",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <div className="w-72">
      <p className="text-sm">商品詳細</p>
      <Separator className="my-3" />
      <div className="flex h-6 items-center gap-3 text-sm">
        <span>数量</span>
        <Separator orientation="vertical" />
        <span>価格</span>
        <Separator orientation="vertical" />
        <span>在庫</span>
      </div>
    </div>
  ),
};
