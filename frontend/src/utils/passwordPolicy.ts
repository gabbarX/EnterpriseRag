import type { FormRule } from 'tdesign-vue-next'

export const PASSWORD_SPECIAL_CHARS = '!@#$%^&*()_+-=[]{}|;:,.<>?'
export const PASSWORD_SPECIAL_CHAR_REGEX = /[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]/

export function newPasswordRules(
  complexEnabled: boolean,
  extra: FormRule[] = [],
): FormRule[] {
  const common: FormRule[] = [
    { required: true, message: 'Enter password', type: 'error' },
    { min: 8, message: 'Password must be at least 8 characters', type: 'error' },
    { max: 32, message: 'Password cannot exceed 32 characters', type: 'error' },
  ]
  if (complexEnabled) {
    return [
      ...common,
      { pattern: /[a-z]/, message: 'Password must contain lowercase letters', type: 'error' },
      { pattern: /[A-Z]/, message: 'Password must contain uppercase letters', type: 'error' },
      { pattern: /\d/, message: 'Password must contain numbers', type: 'error' },
      {
        pattern: PASSWORD_SPECIAL_CHAR_REGEX,
        message: `Password must contain special characters: ${PASSWORD_SPECIAL_CHARS}`,
        type: 'error',
      },
      ...extra,
    ]
  }
  return [
    ...common,
    { pattern: /[a-zA-Z]/, message: 'Password must contain letters', type: 'error' },
    { pattern: /\d/, message: 'Password must contain numbers', type: 'error' },
    ...extra,
  ]
}
