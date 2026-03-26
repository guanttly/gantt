<script setup lang="ts">
import type { OrgTreeNode } from '@/api/org'
import { AddOutline, CaretDownOutline, CaretForwardOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { computed, ref, watch } from 'vue'
import { NButton, NIcon, NTag } from 'naive-ui'
import { ORG_NODE_TYPE_LABELS, isProtectedOrgNode } from '@/api/org'

defineOptions({ name: 'OrgTreeBranch' })

const props = defineProps<{
  nodes: OrgTreeNode[]
  filterText: string
  depth?: number
}>()

const emit = defineEmits<{
  add: [node?: OrgTreeNode]
  edit: [node: OrgTreeNode]
  delete: [node: OrgTreeNode]
}>()

function matches(node: OrgTreeNode, keyword: string): boolean {
  const normalized = keyword.trim().toLowerCase()
  if (!normalized)
    return true

  if (node.name.toLowerCase().includes(normalized))
    return true

  if (node.code.toLowerCase().includes(normalized))
    return true

  return (node.children || []).some(child => matches(child, normalized))
}

const visibleNodes = computed(() => {
  return props.nodes.filter(node => matches(node, props.filterText))
})

const currentDepth = computed(() => props.depth ?? 0)
const expandedNodeIds = ref<Set<string>>(new Set())
const isFiltering = computed(() => props.filterText.trim().length > 0)

watch(
  () => props.nodes,
  (nodes) => {
    const nextExpanded = new Set(expandedNodeIds.value)
    for (const node of nodes) {
      if (node.children?.length) {
        nextExpanded.add(node.id)
      }
    }
    expandedNodeIds.value = nextExpanded
  },
  { immediate: true, deep: true },
)

function handleAdd(node?: OrgTreeNode) {
  emit('add', node)
}

function handleEdit(node: OrgTreeNode) {
  emit('edit', node)
}

function handleDelete(node: OrgTreeNode) {
  emit('delete', node)
}

function getNodeTypeLabel(node: OrgTreeNode) {
  return ORG_NODE_TYPE_LABELS[node.node_type]
}

function hasChildren(node: OrgTreeNode) {
  return !!node.children?.length
}

function isExpanded(node: OrgTreeNode) {
  return isFiltering.value || expandedNodeIds.value.has(node.id)
}

function toggleExpanded(node: OrgTreeNode) {
  if (!hasChildren(node) || isFiltering.value) {
    return
  }

  const nextExpanded = new Set(expandedNodeIds.value)
  if (nextExpanded.has(node.id)) {
    nextExpanded.delete(node.id)
  }
  else {
    nextExpanded.add(node.id)
  }
  expandedNodeIds.value = nextExpanded
}
</script>

<template>
  <ul class="org-branch" :class="{ 'is-root': currentDepth === 0 }" v-if="visibleNodes.length">
    <li v-for="node in visibleNodes" :key="node.id" class="org-item" :class="{ 'is-nested': currentDepth > 0 }">
      <div class="org-node-card" :data-depth="Math.min(currentDepth, 2)" :class="{ 'is-child-card': currentDepth > 0 }">
        <div class="org-node-main">
          <button
            class="node-toggle"
            :class="{ 'is-placeholder': !hasChildren(node), 'is-disabled': isFiltering }"
            type="button"
            :tabindex="hasChildren(node) ? 0 : -1"
            :aria-label="isExpanded(node) ? '收起子节点' : '展开子节点'"
            @click.stop="toggleExpanded(node)"
          >
            <n-icon v-if="hasChildren(node)" :size="14">
              <component :is="isExpanded(node) ? CaretDownOutline : CaretForwardOutline" />
            </n-icon>
          </button>
          <span v-if="currentDepth > 0" class="node-branch-elbow" aria-hidden="true"></span>
          <div class="node-meta">
            <div class="node-title-row">
              <span class="node-name" :data-depth="Math.min(currentDepth, 2)">{{ node.name }}</span>
              <n-tag v-if="getNodeTypeLabel(node)" size="small" :bordered="false" type="info" :class="`node-type-tag depth-${Math.min(currentDepth, 2)}`">{{ getNodeTypeLabel(node) }}</n-tag>
            </div>
            <div class="node-code">{{ node.code }}</div>
          </div>
        </div>
        <div class="node-actions">
          <n-button quaternary circle size="small" @click.stop="handleAdd(node)">
            <template #icon>
              <n-icon><AddOutline /></n-icon>
            </template>
          </n-button>
          <n-button quaternary circle size="small" @click.stop="handleEdit(node)">
            <template #icon>
              <n-icon><CreateOutline /></n-icon>
            </template>
          </n-button>
          <n-button v-if="!isProtectedOrgNode(node)" quaternary circle size="small" type="error" @click.stop="handleDelete(node)">
            <template #icon>
              <n-icon><TrashOutline /></n-icon>
            </template>
          </n-button>
        </div>
      </div>

      <OrgTreeBranch
        v-if="node.children?.length && isExpanded(node)"
        :nodes="node.children"
        :filter-text="filterText"
        :depth="currentDepth + 1"
        @add="handleAdd"
        @edit="handleEdit"
        @delete="handleDelete"
      />
    </li>
  </ul>
</template>

<style scoped>
.org-branch {
  margin: 0;
  padding: 0 0 0 34px;
  list-style: none;
  position: relative;
}

.org-branch::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 14px;
  left: 13px;
  width: 1px;
  background: linear-gradient(180deg, rgba(148, 163, 184, 0.34), rgba(148, 163, 184, 0.14));
}

.org-branch.is-root {
  padding-left: 0;
}

.org-branch.is-root::before {
  display: none;
}

.org-item + .org-item {
  margin-top: 14px;
}

.org-item {
  position: relative;
}

.org-node-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 70px;
  padding: 14px 16px;
  border: 1px solid var(--admin-border);
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbfd 100%);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
  transition: border-color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease;
}

.org-node-card[data-depth='0'] {
  border-color: rgba(15, 118, 110, 0.18);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(239, 248, 247, 0.96) 100%);
}

.org-node-card[data-depth='1'] {
  margin-top: 2px;
  border-color: rgba(59, 130, 246, 0.16);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(243, 247, 255, 0.94) 100%);
}

.org-node-card[data-depth='2'] {
  margin-top: 4px;
  border-color: rgba(148, 163, 184, 0.24);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(248, 250, 252, 0.94) 100%);
}

.org-node-card.is-child-card {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96) 0%, rgba(248, 251, 253, 0.88) 100%);
}

.org-node-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.node-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
  color: var(--admin-primary);
  cursor: pointer;
  flex-shrink: 0;
}

.node-toggle.is-placeholder {
  background: transparent;
  cursor: default;
}

.node-toggle.is-disabled {
  color: var(--admin-text-muted);
}

.node-meta {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.node-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
}

.node-name {
  font-weight: 600;
  color: var(--admin-text);
}

.node-name[data-depth='0'] {
  font-size: 19px;
}

.node-name[data-depth='1'] {
  font-size: 17px;
  color: #1d4ed8;
}

.node-name[data-depth='2'] {
  font-size: 16px;
}

.node-type-tag.depth-0 {
  --n-color: rgba(15, 118, 110, 0.12);
  --n-text-color: #0f766e;
}

.node-type-tag.depth-1 {
  --n-color: rgba(59, 130, 246, 0.12);
  --n-text-color: #2563eb;
}

.node-type-tag.depth-2 {
  --n-color: rgba(148, 163, 184, 0.14);
  --n-text-color: #475569;
}

.node-branch-elbow {
  position: relative;
  flex: 0 0 14px;
  width: 14px;
  height: 14px;
}

.node-branch-elbow::before {
  content: '';
  position: absolute;
  top: 50%;
  left: -21px;
  width: 21px;
  height: 1px;
  background: rgba(148, 163, 184, 0.34);
}

.node-branch-elbow::after {
  content: '';
  position: absolute;
  top: calc(50% - 4px);
  left: -2px;
  width: 8px;
  height: 8px;
  border: 1px solid rgba(15, 118, 110, 0.22);
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
}

.node-code {
  color: var(--admin-text-muted);
  font-size: 12px;
  line-height: 1.4;
  font-family: 'JetBrains Mono', 'SFMono-Regular', 'Consolas', monospace;
}

.node-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .org-branch {
    padding-left: 24px;
  }

  .org-branch::before {
    left: 9px;
  }

  .org-node-card {
    align-items: flex-start;
    flex-direction: column;
  }

  .org-node-main {
    width: 100%;
  }

  .node-actions {
    align-self: flex-end;
  }

  .node-branch-elbow::before {
    left: -17px;
    width: 17px;
  }
}
</style>