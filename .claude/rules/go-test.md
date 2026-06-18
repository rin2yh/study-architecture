---
paths:
  - "**/*_test.go"
---

# Go テスト規約

[[test.md]] (テーブル駆動・ケース分類など) に加え、Go 固有の書き方をここに集める。

## 構造体で `args` と `want` をまとめる

テーブル駆動のケースを素朴に `wantStatus`, `wantCode`, `wantMessage`, ... と並べると
「どれが引数でどれが期待値か」が読み取りづらい。新規テストでは:

- 入力は **`args` 構造体** にまとめる
- 期待値は **`want` 構造体** にまとめる

雛形:

```go
func TestFoo(t *testing.T) {
    type args struct{ a int; b string }
    type want struct{ status int; body string }
    tests := []struct {
        name string
        args args
        want want
    }{
        {"...", args{1, "x"}, want{200, "ok"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, gotBody := Foo(tt.args.a, tt.args.b)
            if got != tt.want.status { t.Fatalf("status = %d, want %d", got, tt.want.status) }
            if gotBody != tt.want.body { t.Fatalf("body = %q, want %q", gotBody, tt.want.body) }
        })
    }
}
```

引数 1 個 + 期待値 1 個など shape が小さい場合は素のフィールドのままで構わない (構造体に
括る目的は読みやすさで、たかが 1 個に括るのは過剰)。

## 対象の関数 / メソッドごとに `Test` 関数を分ける

1 つの `Test` 関数で複数の関数や挙動を兼ねない。テーブル駆動するときも **テスト対象の
関数ごとに `TestXxx` を立てて、その中で table を回す**。同じ shape の handler 3 つを
1 つの `TestErrorHandlers` に詰め込むのは避ける。理由:

- 失敗時に「どの関数の・どのケースが落ちたか」をテスト名から一発で読み取れる
- カバレッジ計算と coverpkg のスコープが綺麗
- 新規ケース追加で関数間の table を触る必要が無くなる
