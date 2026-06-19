export * from "./gen/shipping/shipping/shipping";
export * from "./gen/shipping/shipping/shipping.zod";
export * from "./gen/shipping/system/system";
export * from "./gen/shipping/system/system.zod";
export * from "./gen/shipping/model";

// default レスポンス由来で各 tag ファイル (system / shipping) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./gen/shipping/system/system";
