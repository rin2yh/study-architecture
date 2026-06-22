import { Alert, AlertDescription, AlertTitle } from "@/shared/ui/alert";

export function HomeErrorBoundary({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-2xl p-8">
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>
          <p>商品一覧の取得に失敗しました。</p>
          <pre className="mt-2 overflow-x-auto text-xs">{message}</pre>
        </AlertDescription>
      </Alert>
    </div>
  );
}
