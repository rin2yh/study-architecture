import { Form } from "react-router";

export function LogoutButton() {
  return (
    <Form method="post" action="/logout">
      <button
        type="submit"
        className="rounded border border-gray-300 px-3 py-1 text-gray-700 hover:bg-gray-50"
      >
        ログアウト
      </button>
    </Form>
  );
}
