/** Stand wordlist. The phrase is the seed string; the node hashes it with SHA256, not BIP39. */
const WORDS = [
  'alpine', 'amber', 'anchor', 'apple', 'apron', 'arch', 'arrow', 'ash',
  'basin', 'beech', 'birch', 'bloom', 'bolt', 'bridge', 'brook', 'cedar',
  'cinder', 'cliff', 'cloud', 'cobalt', 'copper', 'coral', 'creek', 'crest',
  'dawn', 'delta', 'drift', 'dune', 'ember', 'fern', 'field', 'flint',
  'frost', 'garden', 'glade', 'glen', 'granite', 'grove', 'harbor', 'haze',
  'heath', 'hill', 'honey', 'horizon', 'iris', 'ivory', 'jade', 'juniper',
  'kelp', 'lagoon', 'lantern', 'larch', 'ledge', 'lilac', 'linen', 'lodge',
  'maple', 'marble', 'marsh', 'meadow', 'mist', 'moss', 'north', 'oak',
  'olive', 'onyx', 'orchid', 'otter', 'pebble', 'pine', 'plaza', 'pond',
  'quartz', 'quill', 'rain', 'reed', 'ridge', 'river', 'robin', 'rock',
  'sage', 'shore', 'silver', 'slate', 'snow', 'spruce', 'stone', 'storm',
  'summit', 'tide', 'timber', 'trail', 'vale', 'violet', 'willow', 'wind',
] as const

export function normalizePhrase(raw: string): string {
  return raw.trim().toLowerCase().split(/\s+/).filter(Boolean).join(' ')
}

export function generatePhrase(count = 12): string {
  const pool = [...WORDS]
  const out: string[] = []
  while (out.length < count && pool.length > 0) {
    const buf = new Uint32Array(1)
    crypto.getRandomValues(buf)
    const idx = buf[0]! % pool.length
    const word = pool.splice(idx, 1)[0]
    if (word) out.push(word)
  }
  return out.join(' ')
}

export function phrasesMatch(a: string, b: string): boolean {
  return normalizePhrase(a) === normalizePhrase(b) && normalizePhrase(a).length > 0
}
