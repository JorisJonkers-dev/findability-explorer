import { describe, expect, it } from 'vitest'
import { inLanguage, languageChoices, ofLanguage, parseLanguage, withLanguage } from './languages'

describe('language filters', () => {
  it('offers only the languages that appear, Dutch, German and English first', () => {
    expect(languageChoices(['en-US', 'fr', 'en', 'de', 'und', 'nl', 'de'])).toEqual([
      { value: 'nl', label: 'Dutch' },
      { value: 'de', label: 'German' },
      { value: 'en', label: 'English' },
      { value: 'fr', label: 'French' },
      { value: 'und', label: 'Unknown language' },
    ])
    expect(languageChoices([])).toEqual([])
  })

  it('shows a row in the chosen language, American English as English', () => {
    expect(inLanguage('en-US', 'en')).toBe(true)
    expect(inLanguage('de', 'en')).toBe(false)
    expect(inLanguage('de', null)).toBe(true)
    const rows = [{ language: 'nl' }, { language: 'en-US' }, { language: 'en' }]
    expect(ofLanguage(rows, 'en')).toEqual([{ language: 'en-US' }, { language: 'en' }])
    expect(ofLanguage(rows, null)).toEqual(rows)
  })

  it('reads the language from the address, dropping what is no language code', () => {
    expect(parseLanguage('de')).toBe('de')
    expect(parseLanguage('en-US')).toBe('en')
    expect(parseLanguage(['nl', 'de'])).toBe('nl')
    expect(parseLanguage('German')).toBeNull()
    expect(parseLanguage(undefined)).toBeNull()
  })

  it('keeps the language in the address only when one is chosen', () => {
    expect(withLanguage({ status: 'accepted' }, 'de')).toEqual({ status: 'accepted', language: 'de' })
    expect(withLanguage({ status: 'accepted', language: 'de' }, null)).toEqual({ status: 'accepted' })
  })
})
