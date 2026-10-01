# Units Desk — 带量纲与温度类型的表达式求值器

普通单位换算器把摄氏温度当作可乘的普通数：`20 °C × 2` 会得到一个貌似合理却
毫无物理意义的结果。本项目把**绝对温度**（K / °C / °F）与**温差**
（dK / dC / dF）建模为不同的类型，配合长度、质量、时间的量纲向量，在求值时
做类型检查，让无意义的运算直接报错并精确定位。

- `units/` — Go HTTP 服务：词法/语法分析、类型检查、有理数精确求值
- `desk/` — Vue 3 单页应用：编辑表达式、展示逐步推导、高亮错误区间
- `docker-compose.yml` — 分别构建并启动 `units` 与 `desk`

## 快速开始

```bash
# 本地开发
cd units && go test ./... && go run ./cmd/units -addr :8080
cd desk && npm install && npm run dev        # http://localhost:5173

# Docker Compose（分别启动两个服务）
docker compose up --build                    # desk: http://localhost:8088, units: :8080
```

## 表达式语法

```
expr    := add (("in" | "to") unit)?
add     := mul (("+" | "-") mul)*
mul     := unary (("*" | "/") unary)*
unary   := "-" unary | primary
primary := number unit? | unit | "(" expr ")"
number  := 整数 | 小数 | 有理数 "a/b"（斜杠两侧不得有空格）
unit    := m | cm | kg | g | s | min | K | C | F | dK | dC | dF
```

`°C`、`°F` 中的度号可选。裸单位表示 1 个该单位（如 `m` ≡ `1 m`）。

## 类型与量纲规则

所有量都以**约分有理数**（`math/big.Rat`）+ 量纲向量 `(L, M, T, Θ)` 表示，
基准单位为 m、kg、s、K。温度量额外携带种类：

| 种类 | 单位 | 说明 |
|---|---|---|
| `absolute` | K, C, F | 温标上的一个点 |
| `delta` | dK, dC, dF | 温度差（区间） |
| `regular` | 其余 | 普通量（含无量纲数） |

温标换算（精确有理数）：

```
K = C + 27315/100
K = (F − 32) × 5/9 + 27315/100        （华氏温差按 5/9 换成开尔文温差）
```

运算规则：

| 运算 | 结果 |
|---|---|
| 绝对 − 绝对 | 温差 |
| 绝对 ± 温差、温差 + 绝对 | 绝对 |
| 温差 − 绝对 | **非法** |
| 绝对 + 绝对 | **非法** |
| 绝对 ×/÷ 任何量 | **非法** |
| 其余 +/− | 量纲与种类必须相容，否则**非法** |
| 温差 ×/÷ 普通量 | 合法，结果为普通量 |

换算目标 `expr in <unit>` 要求量纲相同且种类匹配
（绝对温度不能换算到温差单位，反之亦然）。

## HTTP API

`POST /api/eval`，请求体 `{"expression": "20 C + 5 dC in F"}`。

成功（200）：

```json
{
  "display": "5963/20 K",
  "baseKind": "absolute", "baseDimText": "Θ",
  "targetText": "77 F",
  "steps": [ { "nodeType": "number", "source": "20 C", "kind": "absolute",
               "dimText": "Θ", "value": {"fraction": "5863/20", ...}, ... } ]
}
```

失败（400），错误定位到**最小**表达式区间（rune 索引）：

```json
{ "error": "absolute temperatures cannot be multiplied",
  "kind": "type", "srcStart": 0, "srcEnd": 8,
  "locator": "20 C * 2\n^^^^^^^^" }
```

`steps` 按后序给出每个语法节点的类型（absolute/delta/regular）、量纲与
基准单位精确值，前端据此展示逐步推导。

### `decimal` 字段的边界

`decimal` 是有理数的十进制展开，但任意有理数的展开可以任意长
（如 `1/99999989` 的循环节约 1 亿位），因此它有界：

- 十进制指数在 `[-6, 20]` 内用普通记法：循环节加括号（`0.1(6)`），
  小数最多 100 位，超出以 `…` 结尾；
- 超出该范围用科学记数法，21 位有效数字
  （`1.42857142857142857142…×10⁻¹⁰`），展开恰在窗口内结束时省略 `…`。

因此很大的数不会显示成 `Infinity`，很小的非零数不会显示成 `0`，
响应体大小始终有界，摘要与推导步骤中的十进制永远与精确分数一致。

## 页面行为

- 编辑表达式或切换目标单位后约 300 ms 自动重算（回车/按钮/示例立即触发）；
  输入一旦与屏上结果不一致，旧结果与旧错误高亮立即清除，不会滞留。
- 每次求值取消上一个未完成的请求，且只有最新一次请求的响应会被采纳：
  乱序返回的旧响应（数值或错误区间）不会覆盖新输入对应的状态。

## 测试

- **Go 单元测试**：规则矩阵、换算表、错误区间、推导步骤
  （`units/internal/unitcalc/evaluator_test.go` 等）。
- **对拍测试**：随机生成小型语法树，用主求值器与一个独立实现的参考
  求值器交叉验证数值、量纲、种类与非法性判定；同时把树渲染回源码重新
  解析再求值，校验解析器一致性（`refeval_test.go`、`convert_test.go`）。
- **浏览器流程**（Playwright，`desk/e2e/eval.spec.js`）：先算出合法结果，
  再输入非法运算，断言旧结果被清除、错误区间高亮，再恢复合法表达式；
  另覆盖编辑即自动重算、切换目标单位重算，以及慢响应乱序返回时不得
  覆盖更新结果。
  运行：`cd desk && npx playwright test`（需先启动 units 与 vite dev）。
