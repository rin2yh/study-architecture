import { Form } from "react-router";

interface LoginFormProps {
  error?: string;
}

export function LoginForm({ error }: LoginFormProps) {
  return (
    <div className="mx-auto max-w-sm p-8">
      <h1 className="text-3xl font-bold">ログイン</h1>
      <Form method="post" className="mt-6 flex flex-col gap-4">
        <label className="flex flex-col gap-1 text-sm">
          メールアドレス
          <input
            type="email"
            name="email"
            required
            autoComplete="email"
            className="rounded border border-gray-300 px-3 py-2"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          パスワード
          <input
            type="password"
            name="password"
            required
            autoComplete="current-password"
            className="rounded border border-gray-300 px-3 py-2"
          />
        </label>
        {error && (
          <p role="alert" className="text-sm text-red-600">
            {error}
          </p>
        )}
        <button
          type="submit"
          className="rounded bg-gray-900 px-3 py-2 text-white hover:bg-gray-700"
        >
          ログイン
        </button>
      </Form>
    </div>
  );
}
