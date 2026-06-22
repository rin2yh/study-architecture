import type { Meta, StoryObj } from "@storybook/react-vite";
import { Input } from "ui/input";
import { Label } from "ui/label";

const meta: Meta = {
  title: "ui/Label",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <div className="flex w-80 flex-col gap-2">
      <Label htmlFor="email">メールアドレス</Label>
      <Input id="email" placeholder="you@example.com" />
    </div>
  ),
};
