export {}

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $accessOperation: (operation: string) => boolean
  }
}
