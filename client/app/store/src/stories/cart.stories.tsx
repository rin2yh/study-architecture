import type { Meta, StoryObj } from "@storybook/react-vite";
import { CartList } from "@/routes/cart/components/cart-list";
import { EmptyCart } from "@/routes/cart/components/empty-cart";
import { cartItems } from "./_fixtures";
import { renderInRouter } from "./_router";

const meta: Meta = {
  title: "pages/Cart",
};

export default meta;

type Story = StoryObj<typeof meta>;

const noop = () => {};

export const WithItems: Story = {
  render: () => renderInRouter(<CartList items={cartItems} onSetQty={noop} onRemove={noop} />),
};

export const Empty: Story = {
  render: () => renderInRouter(<EmptyCart />),
};
