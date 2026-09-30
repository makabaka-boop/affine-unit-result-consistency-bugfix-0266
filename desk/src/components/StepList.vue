<script setup>
defineProps({
  steps: { type: Array, required: true },
})

const emit = defineEmits(['hover'])

const kindClass = {
  absolute: 'kind-abs',
  delta: 'kind-delta',
  regular: 'kind-reg',
}

function kindBadge(k) {
  return { absolute: '绝对', delta: '温差', regular: '普通' }[k] || k
}

function nodeLabel(s) {
  if (s.nodeType === 'number') return '字面量'
  if (s.nodeType === 'unary') return '一元 −'
  if (s.nodeType === 'convert') return `换算 →`
  return `二元 ${s.op}`
}
</script>

<template>
  <ol class="steps">
    <li
      v-for="(s, i) in steps"
      :key="i"
      class="step"
      @mouseenter="emit('hover', s)"
      @mouseleave="emit('hover', null)"
    >
      <div class="step-head">
        <span class="step-index">#{{ i + 1 }}</span>
        <span class="step-node">{{ nodeLabel(s) }}</span>
        <code class="step-source">{{ s.source }}</code>
        <span class="badge" :class="kindClass[s.kind]">{{ kindBadge(s.kind) }}</span>
        <span class="dim">{{ s.dimText }}</span>
      </div>
      <div class="step-body">
        <span class="step-value" data-testid="step-value">
          {{ s.value.fraction }} <em v-if="s.unitText">{{ s.unitText }}</em>
        </span>
        <span class="step-decimal">= {{ s.value.decimal }}<em v-if="s.unitText"> {{ s.unitText }}</em></span>
        <span class="step-span">[{{ s.srcStart }}, {{ s.srcEnd }})</span>
      </div>
    </li>
  </ol>
</template>
