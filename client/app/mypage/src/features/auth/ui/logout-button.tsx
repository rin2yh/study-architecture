import { Form } from "react-router";
import { Button } from "ui/button";

export function LogoutButton() {
  return (
    <Form method="post" action="/logout">
      <Button type="submit" variant="outline" size="sm">
        ログアウト
      </Button>
    </Form>
  );
}
