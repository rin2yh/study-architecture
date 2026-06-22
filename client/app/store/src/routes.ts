import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home/home.tsx"),
  route("cart", "routes/cart/cart.tsx"),
  route("checkout", "routes/checkout/checkout.tsx"),
] satisfies RouteConfig;
