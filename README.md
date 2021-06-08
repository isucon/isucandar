# isucandar

[![test](https://github.com/isucon/isucandar/workflows/test/badge.svg)](https://github.com/isucon/isucandar/actions?query=workflow%3Atest)
[![codecov](https://codecov.io/gh/isucon/isucandar/branch/master/graph/badge.svg?token=KO1N8H5S53)](https://codecov.io/gh/isucon/isucandar)

isucandar は [ISUCON](http://isucon.net/) などの負荷試験で使える機能を集めたベンチマーカーフレームワークです。

主な機能として、ブラウザのように振る舞うエージェント、複数階層のスタックトレースを持ったエラー、スコア計算、並列数を制御しつつ外部から停止可能なワーカーなどがあります。

## 使い方

### agent

`isucandar/agent` はブラウザに近い(似せた)挙動をすることを目的として作られたパッケージです。

`net/http` を基礎にしつつ、いくつかの拡張が行われています。

```golang
//// Agent
// NewAgent の引数には可変長で func(*Agent) error な関数を渡せます。
// その中で Agent の初期設定を完了させてください。
// 簡易につかえるように、いくつかの AgentOption を返す関数が用意されています。
agent, err := NewAgent(WithBaseURL("http://isucon.net"))

// 通常の http.NewRequest のように呼び出せます。
req, err := agent.NewRequest(http.MethodGet, "/", nil)
// あるいは、以下のような形でも作成できます。
// req, _ := agent.GET("/")
// req, _ := agent.POST("/", body)

// タイムアウトの制御は主に Context.WithTimeout で行われることを想定しています。
// 利用している Transport や Dialer は DefaultTransport や DefaultDialer を参照してください。
res, err := agent.Do(context.TODO(), req)
// Agent は自動的に以下のような挙動で振る舞います。
// - CookieJar を持っているので Cookie を保存している
// - CacheStore を利用して Conditinal GET あるいはキャッシュ残存期間次第では、リクエストを行わずキャッシュからレスポンスを復元する
// - Content-Encoding で gzip, deflate, brotli のいずれかが指定されて居た場合、自動的に展開する(Accept-Encoding も付与します)
// - 自身の Name に応じて User-Agent を設定する
// - 特に Accept が指定されていない時、自動でブラウザの送るような Accept を送信する

// 取得した HTTP レスポンスを使って、さらにブラウザのような挙動をさせることができます。
resources, err := agent.ProcessHTML(context.TODO(), req, req.Body)
// Agent は HTML を解析し、以下のようなルールに従って追加のリソースへリクエストを送信します。
// - script, link 要素で収集対象となるもの(src が設定されている、 rel が stylesheet である、など)を取得します
// - img 要素も収集しますが、ブラウザの挙動に従い、 loading="lazy" なものは無視します
// - script 要素の async / defer は考慮しません
// 挙動の参考としては『HTML をロードしてから onload が実行されるまでに発行されるリクエスト』を基準としています。
// リクエストの順序などは考慮されていません。
// 厳密な挙動が必要な場合は、外部で実装してください。

//// CacheStore
// Agent は CacheStore を持ち、それを利用してブラウザに似せた Conditinal GET や、
// キャッシュを利用して、メモリからレスポンスを復元したりします。
// もし Cache が必要ないようであれば、 WithNoCache() を NewAgent の引数へ渡してください。
agent, _ := NewAgent(WithNoCache())

// また、なんらかの理由でキャッシュをクリアしたくなった場合は agent.CacheStore.Clear() で削除できます。
agent.CacheStore.Clear()
```

#### 補足

- `Agent` を複数のユーザー間で使い回さないでください。 `Agent` は1つの User−Agent として機能するように実装されています。
- `ProcessHTML` は基本的に低速です。すべてのページでこれを利用しようとしてはいけません。チェックに必要な場合のみ利用してください。

### failure

isucandar 独自のエラーや、それらのコレクションを扱うパッケージです。基本的には [xerrors](https://golang.org/x/xerrors) をベースに作成されていますが、以下のような点が異なります。

- 取得数を指定したり、除外設定のできるコールスタック
  - xerrors 標準では1つしかコールスタックを保持できないためです
- 複数個タグのようにつけられるエラーコード

```golang
//// Code
// Code はエラーコードそのものを指す interface です。
// ErrorCode() string と Error() string が実装されていれば満たすことができます。
// 基本的には StringCode を介して定義するのがかんたんです。
var StandardErrorCode failure.StringCode = "standard"

//// Error
// Error はエラーコード、コールスタックの保持などを行う error 互換の構造体です。
err := NewError(StandardErrorCode, fmt.Errorf("original error message"))
// NewError は基本的に渡されたエラーを Code でラッピングしますが、一部のエラーは追加で Code を付与します。
// - net.Error.Timeout() == true: TimeoutErrorCode
// - net.Error.Temporary() == true: TemporaryErrorCode
// - context.Canceled: CanceledErrorCode

// コールスタックを出力したりできます
fmt.Printf("%v", err)
// standard: original error message
fmt.Printf("%+v", err)
// standard:
//     github.com/isucon/isucandar/failure.TestPrint:
//         ~/src/github.com/isucon/isucandar/failure/failure_test.go:10
// - original error message

// 最も最近つけられた ErrorCode は以下のように取得できます。
// Error ではない場合、自動的に UnknownErrorCode の ErrorCode が返ります。
code := GetCode(err)
// => "standard"

// そのエラーに紐付いている ErrorCode をすべて取得します。
// なんの ErrorCode も紐付いていない時は UnknownErrorCode 単体が返ります。
codes := GetCodes(err)
// => []string{"standard"}

// 元のエラーがなんであったかなどは xerrors や errors 同様に判別できます。
Is(err, context.DeadlineExceeded)

//// Backtrace & BacktraceCleaner
// failure はコールスタックを Error に保存しますが、その深度は変数で変更できます。
CaptureBacktraceSize = 1 // default: 5

// BacktraceCleaner は保存される Backtrace から除外するものを指定できます。
// 例えば組み込みパッケージの Backtrace を除外する指定は以下のようにできます。
BacktraceCleaner.Add(SkipGOROOT())

// Backtrace matcher はかんたんに実装できます。
BacktraceCleaner.Add(func(backtrace *Backtrace) bool {
  return strings.HasSuffix(backtrace.File, "_test.go")
})

//// Errors
// Errors は Error の収集と集計を高速かつかんたんに行うための構造体です。
errors := NewErrors(context.TODO())
errors.Add(err)
// NewErrors に渡した Context が終了すると、エラーの収集が終わったことを伝えられます。
// Errors.Add(error) は内部的に別 goroutine で処理しているため、即座には Errors 内部に追加されません。
// その代わり、ロックを気にせず追加し続けることができます。

// 明示的に収集が完了したことを示すために Done してもよいです。
errors.Done()

// 最終的に以下の関数達で集計をすることができます。
// ErrorCode ごとにエラーメッセージを集計します。
// ErrorCode は GetCode(error) で得られたものが採用されます。
errors.Messages() // => map[string][]string

// ErrorCode ごとに数を集計します。
// ErrorCode は GetCodes(err) で得られたすべてのコードに対して加算するため、
// 総数は実際の error の数より大きくなることに注意してください。
errros.Count() // => map[string]int64
// 例えば unknown が 1 以上なら Critical Error とする、などが考えられます。

// すべての error を返却します。
errors.All() // => []error
```

#### 補足

- 集計系の関数(`Messages` / `Count` / `All`)は集計完了前でもいつでも取り出せます。

### score

スコア集計のためのパッケージです。

```golang
//// Score
// スコアを集計、点数の計算までを行います。
score := NewScore(context.TODO())
// Errors 同様、収集の完了を Context 経由で伝えることができます。

// スコアは文字列によってタグ付けされており、各タグに得点を設定できます。
score.Set("success-get", 1)
score.Set("success-post", 5)

// タグを指定して1ずつスコアを加算します。
score.Add("success-get")
score.Add("success-post")

// Done で明示的にスコア収集の完了を伝えられます。
score.Done()

// 以下の関数達で集計結果を得ることができます。
// 各タグの個数を出力します。
score.Breakdown()
// => map[string]int64{ "success-get": 1, "success-post": 1 }

// 合計得点を出力します。
score.Sum()
// => 6

// Done しつつ Sum をします。
score.Total()
// => 6
```

#### 補足

- `Breakdown()` や `Sum()` はいつでも実行できます。ロックなどを外部から考慮する必要はありません。

### worker

同じ処理を複数回実行したり、並列数を抑えながら無限に実行したりする処理の制御を提供します。

```golang
//// Worker
// Worker はオプションによって少し挙動が変わります。
// ループ回数を指定するようなものは以下のように
limitedWorker, err := NewWorker(f, WithLoopCount(5))
// ループ回数を指定しない場合は以下のように作成します。
unlimitedWorker, err := NewWorker(f, WithInfinityLoop())

// 作成時の引数に渡す f が処理される内容です。
f := func(ctx context.Context, i int) {
  // ctx : 渡されてきた Context
  // i : 何回目の実行か。ループ回数の指定がない場合は常に -1 になります。
}

// Worker は任意のタイミングで Context を介して停止が可能です。
// 停止を通知された Worker は新たな実行をせず、なるべく素早く実行を終了します。
ctx, cancel := context.WithTimeout(context.TODO(), 1 * time.Second)
defer cancel()
worker.Process(ctx)
// Process は起動済みのジョブのすべての実行を待ちます。

// 外部から Worker の終了を検出することもできます。
worker.Wait()

// Worker は作成時または後からループ回数を変更できます。
worker, err := NewWorker(f, WithLoopCount(10))
worker.SetLoopCount(20)

// Worker は作成時または後から並列数を変更できます。
worker, err := NewWorker(f, WithMaxParallelism(10)/* あるいは WithUnlimitedParallelism() */)
worker.SetParallelism(20)
worker.AddParallelism(20)
// 並列数の変更は実行中であっても反映されます。
```

#### 補足

- ループ回数のない `Worker` を後から制限付きに変えたりすると思わぬエラーが発生する場合があります。

### parallel

同時実行数を制御しつつ、複数のジョブを実行させる処理を提供します。

```golang
//// Parallel
// 初期化時、あるいはあとから同時実行数を設定できます。
parallel := NewParallel(10)
parallel.SetParallelism(5)
parallel.AddParallelism(5)
// 制限値に 0 以下の値を与えると、並列数の上限を設けません。
// 並列数の変更はジョブの起動中であっても構いません。

// 実行可能になるまで待ってから(列に並ぶ)、ジョブを実行します。
// Context を渡すことができますが、 Context が終了しても Parallel はジョブを自動停止はしません。
// ジョブ側で Context の終了を検知して終了してください。
// ただ、ジョブが列に並んでいる最中に Context が終了した場合、
// Parallel は順番が来てもジョブを起動しません。
parallel.Do(ctx, func(b context.Context) {
  // Do の ctx が func(b) に引き渡されます。
})

// 順番待ちのジョブがいる時に、外部からすべての実行を取りやめたい場合は、
// Close を用いてジョブの実行をキャンセルします。
parallel.Close()

// 実行中、あるいは未来実行するすべてのジョブの完了を待ちたい場合は、
// Wait を利用してください。
parallel.Wait()

// 1度以上実行し、Close して停止した Parallel を再利用するには
// Reset による再初期化が必要です。
parallel.Reset()
```

#### 補足

- 並列数に1を設定すると、 `Wait` 時に不安定な挙動を示す場合があります。

## Author

Sho Kusano <rosylilly@aduca.org>

## License

See [LICENSE](https://github.com/isucon/isucandar/blob/master/LICENSE)
