// 应用全局状态
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  /** 侧边栏折叠状态 */
  const sidebarCollapsed = ref(false)

  /** 当前语言 */
  const locale = ref<'zh-CN' | 'en-US'>('zh-CN')

  /** 全局加载状态 */
  const globalLoading = ref(false)

  // ==================== 排班 UI 状态 ====================

  /** 左侧面板选中标签页 */
  const leftTab = ref<'gantt' | 'graph' | 'table'>('gantt')

  /** 右侧面板（聊天助手）是否展开 */
  const rightPanelExpanded = ref(true)

  // ==================== 方法 ====================

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setLocale(lang: 'zh-CN' | 'en-US') {
    locale.value = lang
    localStorage.setItem('locale', lang)
  }

  function setLoading(loading: boolean) {
    globalLoading.value = loading
  }

  function setTab(tab: 'gantt' | 'graph' | 'table') {
    leftTab.value = tab
  }

  function toggleRightPanel() {
    rightPanelExpanded.value = !rightPanelExpanded.value
  }

  function setRightPanel(expanded: boolean) {
    rightPanelExpanded.value = expanded
  }

  // 从 localStorage 恢复
  const savedLocale = localStorage.getItem('locale')
  if (savedLocale === 'zh-CN' || savedLocale === 'en-US')
    locale.value = savedLocale

  return {
    sidebarCollapsed,
    locale,
    globalLoading,
    leftTab,
    rightPanelExpanded,
    toggleSidebar,
    setLocale,
    setLoading,
    setTab,
    toggleRightPanel,
    setRightPanel,
  }
})
