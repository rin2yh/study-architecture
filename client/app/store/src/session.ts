// TODO(#6): member 認証導入後、SSR ローダで Cookie 検証した会員 ID に差し替える。
// ログイン状態の取得をこの 1 箇所に閉じ込め、#6 ではここだけを実装に置き換える。
export const DEV_MEMBER_ID = 1;

export function getCurrentMemberId(): number {
  return DEV_MEMBER_ID;
}
