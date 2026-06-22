import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "ui/card";

const meta: Meta = {
  title: "ui/Card",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <Card className="w-96">
      <CardHeader>
        <CardTitle>ご注文内容</CardTitle>
        <CardDescription>カート内の商品を確認してください。</CardDescription>
        <CardAction>
          <Button variant="outline" size="sm">
            編集
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        <p className="text-sm">コーヒー豆 200g × 2</p>
      </CardContent>
      <CardFooter>
        <Button className="w-full">購入手続きへ</Button>
      </CardFooter>
    </Card>
  ),
};
