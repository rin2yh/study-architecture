import { cn } from "../lib/utils";

export function PageLoading({ className }: { className?: string }) {
  return (
    <div
      className={cn("mx-auto max-w-2xl p-8 text-muted-foreground", className)}
      role="status"
      aria-live="polite"
    >
      読み込み中…
    </div>
  );
}
