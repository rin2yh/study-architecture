// BFF (React Router の本番サーバ) を OpenTelemetry で計装する preload。`node --import` で
// アプリ本体より先に評価し、http(受信) と undici(global fetch=Go サービス呼び出し) が patch
// されてからサーバが起動するようにする (ADR-[[202606241356]])。
//
// 計装対象は store のみ (#64)。Dockerfile が両 UI へ同じファイルを置くため、宛先 env が無い
// backoffice では OTel を一切 import せず no-op にする。これにより backoffice の image に OTel
// 依存が無くても import 解決で落ちない。送り先は Alloy 固定 (ADR-[[202606241356]])。
const endpoint =
  process.env.OTEL_EXPORTER_OTLP_ENDPOINT ?? process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT;

if (endpoint) {
  await setup();
}

async function setup() {
  const { NodeSDK, tracing } = await import("@opentelemetry/sdk-node");
  const { OTLPTraceExporter } = await import("@opentelemetry/exporter-trace-otlp-grpc");
  const { HttpInstrumentation } = await import("@opentelemetry/instrumentation-http");
  const { UndiciInstrumentation } = await import("@opentelemetry/instrumentation-undici");
  const { ATTR_HTTP_ROUTE } = await import("@opentelemetry/semantic-conventions");
  const { ServerResponse } = await import("node:http");

  const sdk = new NodeSDK({
    // service.name 等は OTEL_* env から取り込む (Go の otelx と同じく resource は env 由来)。
    // AlwaysOn は Go 側 (always_on) と揃える。exporter 接続失敗は OTLP/gRPC が遅延接続で
    // background retry するため致命にしない (ADR-[[202606241356]] graceful degradation)。
    traceExporter: new OTLPTraceExporter(),
    sampler: new tracing.AlwaysOnSampler(),
    instrumentations: [
      new HttpInstrumentation({
        // 静的アセットや RR の内部 fetch ノイズは span にしない (生パスのカーディナリティ回避)。
        ignoreIncomingRequestHook: (req) => isIgnoredPath(pathOf(req.url)),
        // 受信 span をルートテンプレートで命名する (otelgin と同じ方針)。ヘッダ・ボディは
        // 既定どおり載せない = 秘匿情報マスキングの計装段 (ADR-[[202606250141]])。
        applyCustomAttributesOnSpan: (span, request, response) => {
          if (!(response instanceof ServerResponse)) return;
          const route = routeTemplate(request.url);
          span.updateName(`${request.method} ${route}`);
          span.setAttribute(ATTR_HTTP_ROUTE, route);
        },
      }),
      new UndiciInstrumentation(),
    ],
  });

  sdk.start();

  // SIGTERM (compose stop) で残りの span を flush してから落とす。
  process.once("SIGTERM", () => {
    sdk.shutdown().catch((e) => console.warn("otel sdk shutdown failed", e));
  });
}

function pathOf(url) {
  return (url ?? "/").split("?")[0];
}

function isIgnoredPath(path) {
  return (
    path.startsWith("/assets/") ||
    path === "/favicon.ico" ||
    path === "/robots.txt" ||
    path === "/manifest.json"
  );
}

// store のルートは全て静的 (パラメータ無し) なので、query と RR の `.data` サフィックスを
// 落とせば pathname がそのままルートテンプレートになる。将来 :id 等を足すならここで畳む。
function routeTemplate(url) {
  const path = pathOf(url).replace(/\.data$/, "");
  return path === "" ? "/" : path;
}
