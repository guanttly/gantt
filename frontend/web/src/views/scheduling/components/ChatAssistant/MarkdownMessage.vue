<script setup lang="ts">
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import { computed, onMounted, ref } from 'vue'

import 'highlight.js/styles/github.css'

interface Props {
  content?: string
}

const props = withDefaults(defineProps<Props>(), {
  content: '',
})

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>`
      }
      catch (err) {
        console.error('Highlight error:', err)
      }
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  },
})

const renderedContent = computed(() => {
  try {
    const text = typeof props.content === 'string'
      ? props.content
      : String(props.content ?? '')
    if (!text)
      return ''
    return md.render(text)
  }
  catch (err) {
    console.error('Markdown render error:', err)
    const fallback = typeof props.content === 'string'
      ? props.content
      : String(props.content ?? '')
    return `<p>${md.utils.escapeHtml(fallback)}</p>`
  }
})

const contentRef = ref<HTMLDivElement>()

onMounted(() => {
  if (contentRef.value) {
    const links = contentRef.value.querySelectorAll('a')
    links.forEach((link) => {
      link.setAttribute('target', '_blank')
      link.setAttribute('rel', 'noopener noreferrer')
    })
  }
})
</script>

<template>
  <div ref="contentRef" class="markdown-message" v-html="renderedContent" />
</template>

<style scoped lang="scss">
.markdown-message {
  line-height: 1.6;
  word-wrap: break-word;

  :deep(h1),
  :deep(h2),
  :deep(h3),
  :deep(h4),
  :deep(h5),
  :deep(h6) {
    margin-top: 16px;
    margin-bottom: 8px;
    font-weight: 600;
    line-height: 1.25;
    color: #303133 !important;
  }

  :deep(h1) {
    font-size: 1.5em;
    padding-bottom: 8px;
    border-bottom: 1px solid #e5e7eb;
  }

  :deep(h2) {
    font-size: 1.25em;
    padding-bottom: 6px;
    border-bottom: 1px solid #e5e7eb;
  }

  :deep(h3) { font-size: 1.1em; }
  :deep(h4) { font-size: 1em; }

  :deep(p) {
    margin: 8px 0;
    color: #606266 !important;
  }

  :deep(ul),
  :deep(ol) {
    margin: 8px 0;
    padding-left: 24px;
    color: #606266 !important;
  }

  :deep(li) {
    margin: 4px 0;
    color: #606266 !important;
  }

  :deep(ul) { list-style-type: disc; }
  :deep(ol) { list-style-type: decimal; }
  :deep(ul ul),
  :deep(ol ul) { list-style-type: circle; }

  :deep(strong) {
    font-weight: 600;
    color: #303133 !important;
  }

  :deep(em) {
    font-style: italic;
    color: #303133 !important;
  }

  :deep(a) {
    color: #3b82f6 !important;
    text-decoration: none;

    &:hover { text-decoration: underline; }
  }

  :deep(code) {
    background-color: #ebeef5;
    padding: 2px 6px;
    border-radius: 3px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.9em;
    color: #c7254e !important;
  }

  :deep(pre) {
    background-color: #f6f8fa;
    border-radius: 6px;
    padding: 12px;
    overflow-x: auto;
    margin: 12px 0;

    code {
      background-color: transparent;
      padding: 0;
      color: inherit;
      font-size: 0.875em;
    }
  }

  :deep(blockquote) {
    margin: 12px 0;
    padding: 8px 16px;
    border-left: 4px solid #e5e7eb;
    background-color: #f9fafb;
    color: #6b7280;

    p { margin: 4px 0; }
  }

  :deep(table) {
    border-collapse: collapse;
    width: 100%;
    margin: 12px 0;
    font-size: 0.9em;
  }

  :deep(th),
  :deep(td) {
    border: 1px solid #e5e7eb;
    padding: 8px 12px;
    text-align: left;
  }

  :deep(th) {
    background-color: #f3f4f6;
    font-weight: 600;
  }

  :deep(tr:nth-child(even)) { background-color: #f9fafb; }

  :deep(hr) {
    border: none;
    border-top: 1px solid #e5e7eb;
    margin: 16px 0;
  }

  :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
    margin: 8px 0;
  }

  :deep(input[type="checkbox"]) { margin-right: 6px; }
}
</style>
