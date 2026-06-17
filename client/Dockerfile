# syntax=docker/dockerfile:1
#
# 3つの UI(app/*) 共通の単一 Dockerfile。ビルド対象は APP 引数で切替える。
# ビルドコンテキストは client/（pnpm workspace ルート）:
#   docker build -f client/Dockerfile --build-arg APP=store -t ec-store ./client
# 生成コード(package/api/src)はコミット済みのため Docker 内で codegen は行わない。
ARG APP

FROM node:22-slim AS build
ARG APP
RUN corepack enable
WORKDIR /repo
# 依存はワークスペース全体を解決（allowBuilds で esbuild 等のビルドを許可）
COPY . .
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
RUN pnpm --filter "$APP" build

# Nitro node-server の .output は自己完結（追加の node_modules 不要）
FROM node:22-slim AS runtime
ARG APP
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
COPY --from=build /repo/app/${APP}/.output ./.output
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
