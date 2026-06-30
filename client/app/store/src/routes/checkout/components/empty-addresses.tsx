import { Link } from "react-router";

import { Button } from "ui/button";

export function EmptyAddresses() {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">チェックアウト</h1>
      <p className="mt-6 text-muted-foreground">
        配送先が登録されていません。住所を登録してから注文してください。
      </p>
      <Button asChild variant="link" className="mt-4 px-0">
        <Link to="/">商品一覧へ</Link>
      </Button>
    </div>
  );
}
