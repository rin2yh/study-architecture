export * from "./gen/product/product/product";
export * from "./gen/product/product/product.zod";
export * from "./gen/product/system/system";
export * from "./gen/product/system/system.zod";
export * from "./gen/product/model";

// default レスポンス由来で各 tag ファイル (system / product) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./gen/product/system/system";
