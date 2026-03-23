declare interface WsMessage {
  state: boolean
  MessageType: string
  errorCode: number
  data: WsMessageData | null | undefined
}

declare interface WsMessageData {
  IsJsMonitor: boolean
  MonitorSN: string
  CategoryType: number
  Speed: number
  ParameterData: WsMessageParameter | null | undefined
  ToClientNo: number
  PlatformTaskID: number
}

declare interface WsMessageParameter {
  ProcessType: number
  ProgramIndex: number
  ProgramCount: number
  CurrentIndex: number
  TotalCount: number
  IsFirst: boolean
}

export type {
  WsMessage,
  WsMessageData,
  WsMessageParameter,
}
