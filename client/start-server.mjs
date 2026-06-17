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
import { createReadStream, readdirSync, statSync } from "node:fs";
import { extname, posix, resolve } from "node:path";
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

// dist/client/ は Docker container 内で immutable のため、起動時に 1 度だけ走査して
// pathname → {file, size, mime} の Map を作る。リクエスト時は同期 stat せず Map
// lookup だけで応答する。
function buildStaticMap(dir, map, urlBase = "/") {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = resolve(dir, entry.name);
    const urlPath = posix.join(urlBase, entry.name);
    if (entry.isDirectory()) {
      buildStaticMap(fullPath, map, urlPath);
    } else if (entry.isFile()) {
      map.set(urlPath, {
        file: fullPath,
        size: statSync(fullPath).size,
        mime: MIME[extname(entry.name).toLowerCase()] ?? "application/octet-stream",
      });
    }
  }
  return map;
}

const STATIC = buildStaticMap(CLIENT_DIR, new Map());

function tryServeStatic(req, res) {
  if (req.method !== "GET" && req.method !== "HEAD") return false;
  const pathname = decodeURIComponent(new URL(req.url, "http://_").pathname);
  const entry = STATIC.get(pathname);
  if (!entry) return false;
  res.statusCode = 200;
  res.setHeader("Content-Type", entry.mime);
  res.setHeader("Content-Length", entry.size);
  // Vite/TanStack Start の hashed asset は不変として長期キャッシュ可能。規約 (/assets/*)
  // は build 側の事実で、Map 構築時ではなく serve 時に判定して結合を弱める。
  if (pathname.startsWith("/assets/")) {
    res.setHeader("Cache-Control", "public, max-age=31536000, immutable");
  }
  if (req.method === "HEAD") {
    res.end();
  } else {
    createReadStream(entry.file).pipe(res);
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
