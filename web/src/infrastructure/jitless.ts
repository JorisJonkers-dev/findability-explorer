import * as z from 'zod'

// The server's Content-Security-Policy allows no eval, and zod probes for it with new Function the
// moment it builds an object schema. The generated schemas are built as their module loads, so
// this module has to run first: main.ts imports it before anything that reaches the API.
z.config({ jitless: true })
