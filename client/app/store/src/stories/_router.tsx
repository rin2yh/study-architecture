import type { ReactElement } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";

// ページの表示部品は Link / Form を使うため。
export function renderInRouter(element: ReactElement) {
  const router = createMemoryRouter([{ path: "/", element }]);
  return <RouterProvider router={router} />;
}
