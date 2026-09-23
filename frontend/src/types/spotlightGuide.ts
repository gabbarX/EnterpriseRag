export type GuidePlacement = 'right' | 'left' | 'bottom' | 'top'

export interface SpotlightGuideStep {
  key: string
  target?: string
  placement?: GuidePlacement
  before?: () => void | Promise<void>
  optional?: boolean
  interact?: boolean
}
