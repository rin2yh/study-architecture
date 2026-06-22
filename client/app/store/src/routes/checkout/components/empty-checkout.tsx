import { Link } from "react-router";

import { Button } from "@/shared/ui/button";

export function EmptyCheckout() {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">チェックアウト</h1>
      <p className="mt-6 text-muted-foreground">カートが空です。</p>
      <Button asChild variant="link" className="mt-4 px-0">
        <Link to="/">商品一覧へ</Link>
      </Button>
    </div>
  );
}
