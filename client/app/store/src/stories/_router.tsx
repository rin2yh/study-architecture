import type { ReactElement } from "react";
import { createRoutesStub } from "react-router";

// ページの表示部品は Link / Form を使うため。
export function renderInRouter(element: ReactElement) {
  const Stub = createRoutesStub([{ path: "/", Component: () => element }]);
  return <Stub />;
}
