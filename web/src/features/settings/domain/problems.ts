import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'

/** What was being done when a request failed, which decides how its problem reads. */
export type SettingsAction = 'load' | 'add' | 'test' | 'remove' | 'models' | 'save' | 'turnOff' | 'usage'

export const FORBIDDEN_MESSAGE = 'Only administrators can change these settings.'

const UNPROCESSABLE: Record<SettingsAction, string> = {
  load: 'The settings could not be read.',
  add: 'The provider refused this key.',
  test: 'This key can no longer be read. Remove it and add it again.',
  remove: 'This key cannot be removed here.',
  models: 'This key can no longer be read. Remove it and add it again.',
  save: 'This key can no longer be read. Choose another key.',
  turnOff: 'Product matching could not be turned off.',
  usage: 'This key can no longer be read. Remove it and add it again.',
}

function status(error: unknown): number | null {
  const problem = zProblem.safeParse(error)
  return problem.success ? problem.data.status : null
}

/** Whether the server refused because the account is not an administrator. */
export function isForbidden(error: unknown): boolean {
  return status(error) === 403
}

/** A failed request's problem in words, for the action it failed. */
export function settingsProblem(error: unknown, action: SettingsAction): string {
  const problem = zProblem.safeParse(error)
  if (!problem.success) return describeError(error)
  const { detail } = problem.data
  switch (problem.data.status) {
    case 403:
      return FORBIDDEN_MESSAGE
    case 404:
      return 'That key no longer exists. Reload the page.'
    case 409:
      return detail ?? 'Someone changed these settings at the same time. Reload the page and try again.'
    case 422:
      return detail ? `${UNPROCESSABLE[action]} ${detail}` : UNPROCESSABLE[action]
    case 502:
      return `The provider did not answer${detail ? `: ${detail}` : '.'} Try again in a minute.`
    default:
      return describeError(error)
  }
}
