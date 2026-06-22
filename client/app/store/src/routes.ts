import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home/route.tsx"),
  route("cart", "routes/cart/route.tsx"),
  route("checkout", "routes/checkout/route.tsx"),
] satisfies RouteConfig;
