import type { Meta, StoryObj } from "@storybook/react-vite";
import { Input } from "ui/input";

const meta: Meta = {
  title: "ui/Input",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <div className="flex w-80 flex-col gap-3">
      <Input defaultValue="山田 太郎" />
      <Input placeholder="メールアドレス" />
      <Input disabled placeholder="入力できません" />
      <Input aria-invalid defaultValue="不正な値" />
    </div>
  ),
};
