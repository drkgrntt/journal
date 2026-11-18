// Rating color ramp shared by pattern components. Ratings don't carry a color
// of their own, so the low->high CSS vars are mapped onto the Rating table's
// sorted values (not a hardcoded 1-5). getCssValue is the global from utils/index.js.
const RATING_COLOR_VARS = ['--awful-color', '--bad-color', '--fine-color', '--good-color', '--great-color']

function hexToHsl(hex) {
  if (typeof hex !== 'string') return null
  hex = hex.trim().replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16) / 255
  const g = parseInt(hex.substring(2, 4), 16) / 255
  const b = parseInt(hex.substring(4, 6), 16) / 255
  if (isNaN(r) || isNaN(g) || isNaN(b)) return null
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  let h, s
  const l = (max + min) / 2
  if (max === min) {
    h = s = 0
  } else {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    switch (max) {
      case r: h = (g - b) / d + (g < b ? 6 : 0); break
      case g: h = (b - r) / d + 2; break
      case b: h = (r - g) / d + 4; break
    }
    h *= 60
  }
  return { h, s: s * 100, l: l * 100 }
}

function lerp(a, b, t) {
  return a + (b - a) * t
}

// Returns a function mapping an average rating value to a CSS color.
export function makeRatingColorizer(ratings) {
  const sorted = [...(ratings || [])].sort((a, b) => a.value - b.value)
  const colorsByValue = {}
  sorted.forEach(function(rating, index) {
    colorsByValue[rating.value] = getCssValue(RATING_COLOR_VARS[Math.min(index, RATING_COLOR_VARS.length - 1)]).trim()
  })
  const min = sorted.length ? sorted[0].value : 1
  const max = sorted.length ? sorted[sorted.length - 1].value : 5

  return function colorForRating(value) {
    const clamped = Math.max(min, Math.min(max, value))
    const lower = Math.floor(clamped)
    const upper = Math.ceil(clamped)
    if (lower === upper) {
      return colorsByValue[lower]
    }
    const t = clamped - lower
    const c1 = hexToHsl(colorsByValue[lower])
    const c2 = hexToHsl(colorsByValue[upper])
    // Non-hex CSS values can't be interpolated; fall back to the nearer stop.
    if (!c1 || !c2) return colorsByValue[t < 0.5 ? lower : upper]
    return `hsl(${lerp(c1.h, c2.h, t)}, ${lerp(c1.s, c2.s, t)}%, ${lerp(c1.l, c2.l, t)}%)`
  }
}
