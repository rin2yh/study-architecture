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
COPY . .
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
RUN pnpm --filter "$APP" build
# 対象 app の production deps のみを含む独立した node_modules を /deploy に出す
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm --filter "$APP" --prod --legacy deploy /deploy

# react-router build (build/client + build/server/index.js) を @react-router/serve で listen する。
FROM node:22-slim AS runtime
ARG APP
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0
COPY --from=build /deploy/node_modules ./node_modules
COPY --from=build /repo/app/${APP}/build ./build
# OTel 計装はアプリ本体より先に読ませる必要があるため preload する (#64)。OTLP 宛先 env が無い
# 構成 (backoffice 等) では instrument.mjs が OTel を一切 import せず no-op になる。
COPY --from=build /repo/instrument.mjs ./instrument.mjs
ENV NODE_OPTIONS=--import=file:///app/instrument.mjs
USER node
EXPOSE 3000
CMD ["node_modules/.bin/react-router-serve", "./build/server/index.js"]
