export function OrderHistoryErrorBoundary({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-3xl p-8" role="alert">
      <h1 className="text-3xl font-bold">エラーが発生しました</h1>
      <p className="mt-4 text-red-600">注文履歴の取得に失敗しました。</p>
      <pre className="mt-4 overflow-x-auto rounded bg-gray-100 p-3 text-xs text-gray-700">
        {message}
      </pre>
    </div>
  );
}
