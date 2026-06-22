import type { Meta, StoryObj } from "@storybook/react-vite";
import { CheckoutForm } from "@/features/checkout";
import { EmptyCheckout } from "@/routes/checkout/components/empty-checkout";
import { OrderConfirmed } from "@/routes/checkout/components/order-confirmed";
import { cartItems, order } from "./_fixtures";
import { renderInRouter } from "./_router";

const meta: Meta = {
  title: "pages/Checkout",
};

export default meta;

type Story = StoryObj<typeof meta>;

export const Form: Story = {
  render: () => renderInRouter(<CheckoutForm items={cartItems} submitting={false} />),
};

export const WithError: Story = {
  render: () =>
    renderInRouter(
      <CheckoutForm items={cartItems} submitting={false} error="支払い方法を選択してください。" />,
    ),
};

export const Confirmed: Story = {
  render: () => renderInRouter(<OrderConfirmed order={order} />),
};

export const Empty: Story = {
  render: () => renderInRouter(<EmptyCheckout />),
};
