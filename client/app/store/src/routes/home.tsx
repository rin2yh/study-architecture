import { listProducts, ListProductsResponse } from "api/product";

export {
  HomePage as default,
  HomeErrorBoundary as ErrorBoundary,
  HydrateFallback,
} from "@/pages/home";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}
