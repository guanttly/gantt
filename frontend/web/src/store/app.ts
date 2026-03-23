// 系统级别的用户个性化设置，若有则保存，若无则删除该文件
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const isCollapse = ref(false)
  const contentFullScreen = ref(false)
  const showLogo = ref(true)
  const fixedTop = ref(false)
  const showTabs = ref(false)
  const expandOneMenu = ref(true)
  const elementSize = ref('default')
  const lang = ref('zh')
  const theme = ref({
    state: {
      style: 'default',
      primaryColor: '#409eff',
      primaryTextColor: '#409eff',
      menuType: 'side',
    },
  })
  const menuList = ref([])

  // UI 相关状态 (从 ui store 迁移)
  // 左侧面板当前选中的标签页
  const leftTab = ref<'gantt' | 'graph' | 'table'>('gantt')

  // 右侧面板是否展开
  const rightPanelExpanded = ref(true)

  // 设置左侧标签页
  function setTab(tab: 'gantt' | 'graph' | 'table') {
    leftTab.value = tab
  }

  // 切换右侧面板展开状态
  function toggleRightPanel() {
    rightPanelExpanded.value = !rightPanelExpanded.value
  }

  // 设置右侧面板展开状态
  function setRightPanel(expanded: boolean) {
    rightPanelExpanded.value = expanded
  }

  return {
    isCollapse,
    contentFullScreen,
    showLogo,
    fixedTop,
    showTabs,
    expandOneMenu,
    elementSize,
    lang,
    theme,
    menuList,
    // UI 相关方法和状态
    leftTab,
    rightPanelExpanded,
    setTab,
    toggleRightPanel,
    setRightPanel,
  }
})
