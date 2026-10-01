<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import StepList from './components/StepList.vue'
import HighlightedExpr from './components/HighlightedExpr.vue'

const expression = ref('2 m + 30 cm')
const target = ref('')
const result = ref(null)
const error = ref(null)
const loading = ref(false)
const lastExpr = ref('')
const hoverSpan = ref(null)

const units = ['', 'm', 'cm', 'kg', 'g', 's', 'min', 'K', 'C', 'F', 'dK', 'dC', 'dF']

const examples = [
  '2 m + 30 cm',
  '1 kg + 500 g in g',
  '1 min + 30 s',
  '20 C + 5 dC in F',
  '30 C - 20 C in dF',
  '68 F in C',
  '20 C * 2',
  '20 C + 30 C',
  '5 dC - 20 C',
  '1 m + 1 kg',
]

const sentExpression = computed(() => {
  const e = expression.value.trim()
  if (!target.value) return e
  if (/\b(in|to)\b/.test(e)) return e
  return `${e} in ${target.value}`
})

// The span to highlight in the expression: the error span when there is an
// error, otherwise the hovered derivation step's span.
const activeSpan = computed(() => {
  if (error.value) return [error.value.srcStart, error.value.srcEnd]
  if (hoverSpan.value) return hoverSpan.value
  return null
})

// Out-of-order guard. Every evaluation gets a fresh sequence number and
// aborts the previous in-flight request, so a slower earlier response can
// never overwrite the state belonging to a newer edit: result, error and
// lastExpr on screen always come from the same, latest request.
let requestSeq = 0
let inFlight = null
let debounceTimer = null

async function evaluate() {
  clearTimeout(debounceTimer)
  const expr = sentExpression.value
  const seq = ++requestSeq
  // Clear previous state up-front: an illegal expression must never leave a
  // stale result on screen.
  result.value = null
  error.value = null
  hoverSpan.value = null
  lastExpr.value = expr
  if (inFlight) {
    inFlight.abort()
    inFlight = null
  }
  if (!expr) {
    loading.value = false
    return
  }
  const ctrl = new AbortController()
  inFlight = ctrl
  loading.value = true
  try {
    const resp = await fetch('/api/eval', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ expression: expr }),
      signal: ctrl.signal,
    })
    const body = await resp.json()
    if (seq !== requestSeq) return // a newer evaluation owns the screen
    if (!resp.ok) {
      error.value = body
    } else {
      result.value = body
    }
  } catch (e) {
    if (ctrl.signal.aborted || seq !== requestSeq) return // superseded
    error.value = {
      error: `无法连接 units 服务：${e.message}`,
      kind: 'network',
      srcStart: 0,
      srcEnd: expr.length,
      locator: '',
    }
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

// Auto re-evaluate shortly after the expression or the target unit changes.
// The old result/error is cleared the moment the input diverges from what
// was last evaluated, so stale values or error highlights never sit next to
// the new input looking current.
watch(sentExpression, expr => {
  if (expr === lastExpr.value) return
  result.value = null
  error.value = null
  hoverSpan.value = null
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    if (sentExpression.value !== lastExpr.value) evaluate()
  }, 300)
})

onMounted(evaluate)

function useExample(ex) {
  expression.value = ex
  target.value = ''
  evaluate()
}

function onStepHover(step) {
  hoverSpan.value = step ? [step.srcStart, step.srcEnd] : null
}
</script>

<template>
  <main class="page">
    <h1>Units Desk</h1>
    <p class="tagline">
      带量纲与温度类型的表达式求值 —— 绝对温度不可乘除，摄氏/华氏不是普通乘数。
    </p>

    <section class="editor">
      <div class="input-row">
        <input
          id="expr-input"
          v-model="expression"
          type="text"
          placeholder="例如：20 C + 5 dC in F"
          @keyup.enter="evaluate"
        />
        <select id="target-select" v-model="target" aria-label="目标单位">
          <option value="">（目标单位）</option>
          <option v-for="u in units.filter(Boolean)" :key="u" :value="u">{{ u }}</option>
        </select>
        <button id="eval-btn" @click="evaluate">
          {{ loading ? '计算中…' : '求值' }}
        </button>
      </div>

      <div class="examples">
        <button
          v-for="ex in examples"
          :key="ex"
          class="example"
          type="button"
          @click="useExample(ex)"
        >{{ ex }}</button>
      </div>

      <div v-if="lastExpr" class="expr-view" data-testid="expr-view">
        <HighlightedExpr :expr="lastExpr" :span="activeSpan" />
      </div>

      <div class="status-row">
        <span v-if="loading" class="status" data-testid="loading-hint">计算中…</span>
      </div>
    </section>

    <section v-if="error" id="error-panel" class="panel error" data-testid="error-panel">
      <h2>错误（{{ error.kind }}）</h2>
      <p class="error-msg" data-testid="error-msg">{{ error.error }}</p>
      <pre v-if="error.locator" class="locator" data-testid="error-locator">{{ error.locator }}</pre>
    </section>

    <section v-if="result" id="result-panel" class="panel" data-testid="result-panel">
      <h2>结果</h2>
      <div class="result-grid">
        <div class="result-main">
          <div class="value-line">
            <span class="label">精确值（基准单位）</span>
            <span class="value" data-testid="base-value">{{ result.display }}</span>
          </div>
          <div v-if="result.targetText" class="value-line">
            <span class="label">目标单位</span>
            <span class="value target" data-testid="target-value">{{ result.targetText }}</span>
          </div>
          <div class="meta">
            <span>类型：<b data-testid="result-kind">{{ result.baseKind }}</b></span>
            <span>量纲：<b data-testid="result-dim">{{ result.baseDimText }}</b></span>
            <span v-if="result.targetValue">
              十进制：<b>{{ result.targetValue.decimal }}</b>
            </span>
            <span v-else>
              十进制：<b>{{ result.base.decimal }}</b>
            </span>
          </div>
        </div>
      </div>

      <h3>逐步推导</h3>
      <StepList :steps="result.steps" @hover="onStepHover" />
    </section>
  </main>
</template>
