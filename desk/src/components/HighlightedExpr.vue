<script setup>
defineProps({
  expr: { type: String, required: true },
  span: { type: Array, default: null }, // [start, end) in JS string indices
})
</script>

<template>
  <code class="hl-expr">
    <template v-if="span && span[0] >= 0 && span[1] > span[0] && span[1] <= expr.length">
      <span class="hl-pre">{{ expr.slice(0, span[0]) }}</span>
      <mark class="hl-mark">{{ expr.slice(span[0], span[1]) }}</mark>
      <span class="hl-post">{{ expr.slice(span[1]) }}</span>
    </template>
    <template v-else>{{ expr }}</template>
  </code>
</template>
