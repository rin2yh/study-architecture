// orval の override.mutator から呼ばれる共通 fetch 実装（サーバ側ローダ/サーバ関数でのみ使用）。
// 各サービスの baseURL を process.env から前置する。
//
// 生成コード（httpClient: 'fetch'）は戻り値に { data, status, headers } を期待するため、
// その形で返す。
const makeFetch =
  (envVar: string) =>
  async <T>(url: string, init?: RequestInit): Promise<T> => {
    const base = process.env[envVar] ?? "";
    const res = await fetch(`${base}${url}`, {
      ...init,
      headers: { "content-type": "application/json", ...init?.headers },
    });
    const body =
      res.status === 204 || res.status === 205 || res.body === null ? undefined : await res.json();
    if (!res.ok) {
      throw new Error(`request to ${url} failed: ${res.status} ${res.statusText}`);
    }
    return { data: body, status: res.status, headers: res.headers } as T;
  };

export const productFetch = makeFetch("PRODUCT_API_URL");
export const orderFetch = makeFetch("ORDER_API_URL");
export const paymentFetch = makeFetch("PAYMENT_API_URL");
export const memberFetch = makeFetch("MEMBER_API_URL");
export const shippingFetch = makeFetch("SHIPPING_API_URL");
