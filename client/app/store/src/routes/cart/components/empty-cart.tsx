import { Link } from "react-router";

import { Button } from "ui/button";

export function EmptyCart() {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">カート</h1>
      <p className="mt-6 text-muted-foreground">カートは空です。</p>
      <Button asChild variant="link" className="mt-4 px-0">
        <Link to="/">商品一覧へ</Link>
      </Button>
    </div>
  );
}
