import mitt from 'mitt'
import type { WsMessage } from './ws-message'

export const emitter = mitt<Record<string, WsMessage>>()
