import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home/route.tsx"),
  route("login", "routes/login/route.tsx"),
  route("logout", "routes/logout/route.tsx"),
] satisfies RouteConfig;
