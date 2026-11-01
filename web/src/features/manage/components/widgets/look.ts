/**
 * The classes Manage's widgets share, so every form and button looks and sits the same: a field
 * fills its space, and a row of buttons keeps its distance from the field above it.
 */

/** A labelled field: the label above, the control filling the width. */
export const LABEL = 'flex min-w-0 flex-col gap-1.5 text-sm font-medium text-ink-2'
/** A text box, number box, or select that fills its space. */
export const FIELD = 'h-12 w-full min-w-0 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal text-black'
/** A text area that fills its space. */
export const AREA = 'min-h-32 w-full min-w-0 rounded-xl border-2 border-grey bg-white p-3 text-base font-normal text-black'
/** A row of buttons under fields: spaced from them, wrapping on a narrow screen. */
export const ACTIONS = 'mt-2 flex flex-wrap items-center gap-3'

const BUTTON = 'inline-flex min-h-11 cursor-pointer items-center justify-center rounded-full px-5 text-center font-semibold no-underline disabled:cursor-not-allowed disabled:opacity-60'
/** The main action. */
export const PRIMARY = `${BUTTON} border-0 bg-black text-white hover:bg-ink-2 hover:text-white`
/** A second action, or a link drawn as a button. */
export const SECONDARY = `${BUTTON} border-2 border-solid border-silver bg-white text-black hover:border-black hover:text-black`
/** An action that stops or removes something. */
export const DANGER = `${BUTTON} border-2 border-solid border-orange-ink bg-white text-orange-ink hover:bg-orange-pale`
/** A small button in a table row: it wraps in its cell rather than running past the table's edge. */
export const ROW_BUTTON = 'inline-flex min-h-10 cursor-pointer items-center rounded-full border-2 border-solid border-silver bg-white px-3.5 text-sm font-semibold text-black no-underline hover:border-black hover:text-black disabled:cursor-not-allowed disabled:opacity-60'
/** A toggle in a group of choices, pressed or not. */
export const CHOICE = 'inline-flex min-h-11 cursor-pointer items-center rounded-full border-2 border-solid px-4 text-sm font-semibold'
export const CHOICE_ON = 'border-black bg-black text-white'
export const CHOICE_OFF = 'border-silver bg-white text-black hover:border-black'

/** A table that never runs past its box: it wraps its cells and scrolls only as a last resort. */
export const TABLE_BOX = 'w-full min-w-0 overflow-x-auto'
export const TABLE = 'w-full border-collapse text-left text-[15px]'
export const TH = 'border-b-2 border-black px-2 py-2 align-bottom text-sm font-medium text-muted'
export const TD = 'border-b border-grey px-2 py-2.5 align-top break-words'
