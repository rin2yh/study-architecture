export * from "./gen/member/member/member";
export * from "./gen/member/member/member.zod";
export * from "./gen/member/system/system";
export * from "./gen/member/system/system.zod";
export * from "./gen/member/model";

// default レスポンス由来で各 tag ファイル (system / member) が同名の HTTPStatusCode*
// を生成するため、明示 re-export で star-export の重複 (TS2308) を解消する。
export type {
  HTTPStatusCode1xx,
  HTTPStatusCode2xx,
  HTTPStatusCode3xx,
  HTTPStatusCode4xx,
  HTTPStatusCode5xx,
  HTTPStatusCodes,
} from "./gen/member/system/system";
