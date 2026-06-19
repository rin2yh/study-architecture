export * from "./gen/order/order/order";
export * from "./gen/order/order/order.zod";
export * from "./gen/order/system/system";
export * from "./gen/order/system/system.zod";
export * from "./gen/order/model";

// default レスポンス由来で各 tag ファイル (system / order) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./gen/order/system/system";
