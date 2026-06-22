import { Links, Meta, Outlet, Scripts, ScrollRestoration } from "react-router";
import type { Route } from "./+types/root";
import { Alert, AlertDescription, AlertTitle } from "@/shared/ui/alert";
import "./styles.css";

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width,initial-scale=1" />
        <title>store</title>
        <Meta />
        <Links />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  return <Outlet />;
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-2xl p-8">
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>
          <pre className="overflow-x-auto text-xs">{message}</pre>
        </AlertDescription>
      </Alert>
    </div>
  );
}
