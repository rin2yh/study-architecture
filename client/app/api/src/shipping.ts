export * from "./shipping/shipping/shipping";
export * from "./shipping/shipping/shipping.zod";
export * from "./shipping/system/system";
export * from "./shipping/system/system.zod";
export * from "./shipping/model";

// default レスポンス由来で各 tag ファイル (system / shipping) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./shipping/system/system";
