/** Where auth-api signs a user in, coming back to returnTo (a path with its query) after. */
export function loginUrl(returnTo: string): string {
  return `/auth/login?return=${encodeURIComponent(returnTo)}`
}

/** Where the sign-out form posts. */
export const LOGOUT_ACTION = '/auth/logout'
