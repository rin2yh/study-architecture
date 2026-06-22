import type { Meta, StoryObj } from "@storybook/react-vite";
import { createRoutesStub } from "react-router";
import type { Product } from "api/product";
import Home from "@/routes/home/route";
import { products } from "./_fixtures";

const meta: Meta = {
  title: "pages/Home",
};

export default meta;

type Story = StoryObj<typeof meta>;

// createMemoryRouter では loaderData が props に渡らないため (framework route 規約の再現が要る)。
function renderHome(loaderData: Product[]) {
  const Stub = createRoutesStub([{ path: "/", Component: Home, loader: () => loaderData }]);
  return <Stub />;
}

export const WithProducts: Story = {
  render: () => renderHome(products),
};

export const Empty: Story = {
  render: () => renderHome([]),
};
