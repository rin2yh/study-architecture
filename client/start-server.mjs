// 各 UI app (TanStack Start の SSR handler) を Node 標準 http で listen する起動エントリ。
//
// dist/client/ の静的ファイルを直接返し、それ以外は dist/server/server.js の default
// export (= `{ async fetch(req) }` 形式の handler) へ dispatch する。
//
// 背景: nitro/vite plugin は production build に Vite-dev 用 SSR fallback
// (fetch(req,{viteEnv:"ssr"})) を残し、Docker 内で SSR 中に同一プロセスの HTTP を
// self-fetch してデッドロックする (doc/known-issues.md 参照)。本スクリプトは nitro 抜きの
// tanstackStart() build 出力を薄い Node サーバから直接呼び、self-fetch 経路を絶つ。

import { createServer } from "node:http";
import { Readable } from "node:stream";
import { createReadStream, statSync } from "node:fs";
import { extname, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import process from "node:process";

const PORT = Number(process.env.PORT ?? 3000);
const HOST = process.env.HOST ?? "0.0.0.0";
// 相対パスは cwd 起点で解釈する (各 app ディレクトリで起動される想定)。
const CLIENT_DIR = resolve(process.cwd(), process.env.CLIENT_DIR ?? "./dist/client");
const ENTRY = resolve(process.cwd(), process.env.SERVER_ENTRY ?? "./dist/server/server.js");

const handler = (await import(pathToFileURL(ENTRY).href)).default;

const MIME = {
  ".js": "application/javascript; charset=utf-8",
  ".mjs": "application/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".ico": "image/x-icon",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".svg": "image/svg+xml",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".json": "application/json; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
  ".map": "application/json; charset=utf-8",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
};

function tryServeStatic(req, res) {
  if (req.method !== "GET" && req.method !== "HEAD") return false;
  const url = new URL(req.url, "http://_");
  const pathname = decodeURIComponent(url.pathname);
  if (pathname.includes("\0") || pathname.includes("..")) return false;
  const filePath = resolve(CLIENT_DIR, "." + pathname);
  let st;
  try {
    st = statSync(filePath);
  } catch {
    return false;
  }
  if (!st.isFile()) return false;
  const mime = MIME[extname(filePath).toLowerCase()] ?? "application/octet-stream";
  res.statusCode = 200;
  res.setHeader("Content-Type", mime);
  res.setHeader("Content-Length", st.size);
  if (pathname.startsWith("/assets/")) {
    res.setHeader("Cache-Control", "public, max-age=31536000, immutable");
  }
  if (req.method === "HEAD") {
    res.end();
  } else {
    createReadStream(filePath).pipe(res);
  }
  return true;
}

const server = createServer(async (nodeReq, nodeRes) => {
  try {
    if (tryServeStatic(nodeReq, nodeRes)) return;
    const host = nodeReq.headers.host ?? `${HOST}:${PORT}`;
    const url = `http://${host}${nodeReq.url}`;
    const init = { method: nodeReq.method, headers: nodeReq.headers };
    if (nodeReq.method !== "GET" && nodeReq.method !== "HEAD") {
      init.body = Readable.toWeb(nodeReq);
      init.duplex = "half";
    }
    const webReq = new Request(url, init);
    const webRes = await handler.fetch(webReq);
    nodeRes.statusCode = webRes.status;
    webRes.headers.forEach((v, k) => nodeRes.setHeader(k, v));
    if (webRes.body) {
      Readable.fromWeb(webRes.body).pipe(nodeRes);
    } else {
      nodeRes.end();
    }
  } catch (err) {
    console.error("[start-server] handler error:", err);
    if (!nodeRes.headersSent) nodeRes.statusCode = 500;
    nodeRes.end("Internal Server Error");
  }
});

server.listen(PORT, HOST, () => {
  console.log(`Listening on http://${HOST}:${PORT}`);
});
