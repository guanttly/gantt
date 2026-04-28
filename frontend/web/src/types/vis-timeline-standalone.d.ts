declare module 'vis-timeline/standalone' {
  export * from 'vis-timeline'

  export const moment: {
    (inp?: unknown, format?: unknown, strict?: boolean): unknown
    (inp?: unknown, format?: unknown, language?: string, strict?: boolean): unknown
    localeData: (key?: string) => unknown
    updateLocale: (key: string, values: Record<string, unknown>) => void
    defineLocale: (key: string, values: Record<string, unknown>) => void
    locale: (key: string) => void
  }

  export class DataSet<Item = any> {
    constructor(data?: Item[])
    add(data: Item[] | Item): unknown
    clear(): void
    get(id: string | number): Item | null | undefined
  }
}