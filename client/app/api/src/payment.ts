export * from "./payment/payment/payment";
export * from "./payment/payment/payment.zod";
export * from "./payment/system/system";
export * from "./payment/system/system.zod";
export * from "./payment/model";

// default レスポンス由来で各 tag ファイル (system / payment) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./payment/system/system";
