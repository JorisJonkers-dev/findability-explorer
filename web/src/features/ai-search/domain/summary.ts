/** A share as the page writes it: "24%". */
export function pct(share: number): string {
  return `${String(Math.round(share * 100))}%`
}
