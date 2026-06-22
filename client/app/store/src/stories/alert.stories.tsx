import type { Meta, StoryObj } from "@storybook/react-vite";
import { Alert, AlertDescription, AlertTitle } from "ui/alert";

const meta: Meta = {
  title: "ui/Alert",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <div className="flex w-96 flex-col gap-4">
      <Alert>
        <AlertTitle>お知らせ</AlertTitle>
        <AlertDescription>注文を受け付けました。</AlertDescription>
      </Alert>
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>在庫が不足しています。</AlertDescription>
      </Alert>
    </div>
  ),
};
