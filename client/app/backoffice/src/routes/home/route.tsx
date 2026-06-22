import { listProducts, ListProductsResponse } from "api/product";
import { Alert, AlertDescription, AlertTitle } from "ui/alert";
import { PageLoading } from "ui/page-loading";
import type { Route } from "./+types/route";
import { ProductTable } from "./components/product-table";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const products = loaderData;
  return (
    <div className="mx-auto max-w-4xl p-8">
      <h1 className="text-3xl font-bold">商品管理</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        product サービスの ListProducts を一覧表示
      </p>
      <ProductTable products={products} />
      {products.length === 0 && <p className="mt-6 text-muted-foreground">商品がありません。</p>}
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-4xl p-8">
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>
          <p>商品一覧の取得に失敗しました。</p>
          <pre className="overflow-x-auto text-xs">{message}</pre>
        </AlertDescription>
      </Alert>
    </div>
  );
}

export function HydrateFallback() {
  return <PageLoading className="max-w-4xl" />;
}
