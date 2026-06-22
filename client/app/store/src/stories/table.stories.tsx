import type { Meta, StoryObj } from "@storybook/react-vite";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "ui/table";

const meta: Meta = {
  title: "ui/Table",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: () => (
    <Table className="w-96">
      <TableHeader>
        <TableRow>
          <TableHead>商品</TableHead>
          <TableHead>数量</TableHead>
          <TableHead className="text-right">価格</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow>
          <TableCell>コーヒー豆 200g</TableCell>
          <TableCell>2</TableCell>
          <TableCell className="text-right">¥3,600</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>ドリッパー</TableCell>
          <TableCell>1</TableCell>
          <TableCell className="text-right">¥2,200</TableCell>
        </TableRow>
      </TableBody>
    </Table>
  ),
};
