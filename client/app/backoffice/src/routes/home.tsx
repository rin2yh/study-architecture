import { listProducts, ListProductsResponse } from "api/product";

export {
  ProductListPage as default,
  ProductListErrorBoundary as ErrorBoundary,
  ProductListHydrateFallback as HydrateFallback,
} from "@/pages/product-list";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}
