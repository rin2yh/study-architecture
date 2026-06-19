package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open はテストごとに独立した DB を確保し、その接続プールを返す。envVar が指す DSN の
// DB をテンプレートに CREATE DATABASE ... TEMPLATE でクローンし、テスト終了時に DROP する。
//
// なぜ共有 DB に直接つながず毎回クローンするのか: 結合テストは同じ物理 DB の同じ table を
// TRUNCATE/seed するので、共有のままだと並列実行で seed が競合し、CI では -p 1 (逐次) が
// 必須だった。テスト単位で DB を分ければ競合しないので -p 1 を外して並列化できる。
// テンプレートからの CREATE DATABASE は migration を流し直すよりファイルコピーで速い。
func Open(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(envVar)
	if dsn == "" {
		t.Fatalf("%s is required for integration tests", envVar)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse %s: %v", envVar, err)
	}
	template := pathDB(u)
	clone := template + "_t_" + randSuffix(t)

	ctx := context.Background()

	// CREATE/DROP DATABASE は対象 DB への接続中・トランザクション内では実行できないため、
	// 維持用 DB (postgres) への単発接続から発行する。
	admin := withDB(u, "postgres")
	createClone(t, ctx, admin, clone, template)
	// LIFO で pool.Close より後 (= 先に Close) に走るよう、DROP を先に登録する。
	t.Cleanup(func() { dropClone(ctx, admin, clone) })

	pool, err := pgxpool.New(ctx, withDB(u, clone))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping (%s): %v", clone, err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func pathDB(u *url.URL) string {
	if len(u.Path) > 0 {
		return u.Path[1:]
	}
	return u.Path
}

func withDB(u *url.URL, dbname string) string {
	c := *u
	c.Path = "/" + dbname
	return c.String()
}

func randSuffix(t *testing.T) string {
	t.Helper()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b[:])
}

// connectAdmin は CREATE/DROP DATABASE 用の接続を返す。これらの utility 文は extended
// protocol で prepare できないため、simple protocol を既定にした接続を使う。
func connectAdmin(ctx context.Context, dsn string) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return pgx.ConnectConfig(ctx, cfg)
}

func createClone(t *testing.T, ctx context.Context, adminDSN, clone, template string) {
	t.Helper()
	conn, err := connectAdmin(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	defer conn.Close(ctx)
	sql := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{clone}.Sanitize(), pgx.Identifier{template}.Sanitize())
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("create database %s: %v", clone, err)
	}
}

func dropClone(ctx context.Context, adminDSN, clone string) {
	conn, err := connectAdmin(ctx, adminDSN)
	if err != nil {
		return
	}
	defer conn.Close(ctx)
	// WITH (FORCE): pool.Close 後も残る接続があっても落とせるようにする (PG13+)。
	conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{clone}.Sanitize()))
}
