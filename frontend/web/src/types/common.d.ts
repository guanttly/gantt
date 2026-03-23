export interface SearchParams {
  page: number
  limit: number
  [key: string]: any
}

export interface PageData<T> {
  page: number
  total: number
  list: T[]
}
