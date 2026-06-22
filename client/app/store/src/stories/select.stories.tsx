import type { Meta, StoryObj } from "@storybook/react-vite";
import { Select, SelectTrigger, SelectValue } from "ui/select";

const meta: Meta = {
  title: "ui/Select",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <Select>
      <SelectTrigger className="w-56">
        <SelectValue placeholder="お届け先を選択" />
      </SelectTrigger>
    </Select>
  ),
};
