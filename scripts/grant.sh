#!/usr/bin/env bash
# サービスごとの最小権限ロールを作り、自 schema にのみ DML できるよう GRANT する (ADR-[[202606231000]])。
# migrate (table 作成) 済みのスタックに対して流す。冪等なので何度流してもよい。
set -euo pipefail

cd "$(dirname "$0")/.."

# psql は host に無い場合があるので postgres コンテナ内のものを使う。
apply() { docker compose exec -T "$1" psql -U ec -d "$2" -v ON_ERROR_STOP=1; }

apply db-order    ec_order    < scripts/grant/order.sql
apply db-payment  ec_payment  < scripts/grant/payment.sql
apply db-member   ec_member   < scripts/grant/member.sql
apply db-shipping ec_shipping < scripts/grant/shipping.sql
apply db-product  ec_product  < scripts/grant/product.sql
