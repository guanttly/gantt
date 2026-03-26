import type { App } from 'vue'
import { createI18n } from 'vue-i18n'

const messages = Object.fromEntries(
  Object.entries(
    import.meta.glob<{ default: any }>('../../locales/*.y(a)?ml', {
      eager: true,
    }),
  ).map(([key, value]) => {
    const yaml = key.endsWith('.yaml')
    return [key.slice(14, yaml ? -5 : -4), value.default]
  }),
)

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages,
})

export { i18n }

export function installI18n(app: App<Element>) {
  app.use(i18n)
}
