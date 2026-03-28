// 分页通用逻辑
import type { ListParams, ListResponse, PaginatedResponse } from '@/types/api'
import { ref, watch } from 'vue'

interface UsePaginationOptions<T> {
  /** API 请求函数 */
  fetchFn: (params: ListParams) => Promise<ListResponse<T>>
  /** 默认分页大小 */
  pageSize?: number
  /** 是否立即加载 */
  immediate?: boolean
}

function isPaginatedResponse<T>(value: ListResponse<T>): value is PaginatedResponse<T> {
  return !Array.isArray(value)
}

export function usePagination<T>(options: UsePaginationOptions<T>) {
  const { fetchFn, pageSize = 20, immediate = true } = options

  const loading = ref(false)
  const items = ref<T[]>([]) as { value: T[] }
  const total = ref(0)
  const currentPage = ref(1)
  const currentPageSize = ref(pageSize)
  const keyword = ref('')

  async function fetchData() {
    loading.value = true
    try {
      const params: ListParams = {
        page: currentPage.value,
        page_size: currentPageSize.value,
      }
      if (keyword.value)
        params.keyword = keyword.value

      const res = await fetchFn(params)
      if (isPaginatedResponse(res)) {
        items.value = res.items
        total.value = res.total
        return
      }

      items.value = res
      total.value = res.length
    }
    finally {
      loading.value = false
    }
  }

  function handlePageChange(page: number) {
    currentPage.value = page
    fetchData()
  }

  function handleSizeChange(size: number) {
    currentPageSize.value = size
    currentPage.value = 1
    fetchData()
  }

  function handleSearch(kw: string) {
    keyword.value = kw
    currentPage.value = 1
    fetchData()
  }

  function refresh() {
    fetchData()
  }

  // 监听关键字变化自动搜索（可选 debounce）
  watch(keyword, () => {
    currentPage.value = 1
    fetchData()
  })

  if (immediate) {
    fetchData()
  }

  return {
    loading,
    items,
    total,
    currentPage,
    currentPageSize,
    keyword,
    fetchData,
    handlePageChange,
    handleSizeChange,
    handleSearch,
    refresh,
  }
}
