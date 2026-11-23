# Blur mode and reveal

Blur mode (profile setting, stored per device in `localStorage` as `blur-text-<userId>`) blurs sensitive text so no one can read over your shoulder. This note covers how you read blurred text on purpose, and an idea we deferred.

## What shipped

Tap-to-reveal. Any read-only blurred text marked `can-blur can-reveal tabindex="0"` unblurs on click or Enter/Space, and blurs again on the next tap.

- Applies to journal entries (list and view), action items, routines, thankfuls, reminders, and entry templates.
- A reveal is temporary: it re-blurs after 15 seconds (`REVEAL_MS`) and whenever the tab is hidden (`visibilitychange`). It lives only in the DOM, so navigation or an htmx swap resets it too.
- Form inputs and textareas are unchanged: they unblur on focus and re-blur on focusout.
- Logic is in `internal/web/assets/js/utils/index.js` (`setRevealed`, `toggleReveal`); the pointer cursor on blurred, tappable text is in `assets/css/utils/utils.css`.
- The journal list's entry text sits outside the card's link, so a tap reveals without opening the entry.
- New read-only blurred text should get `can-reveal tabindex="0"` too, or it can only be read by turning blur off.

## Deferred: "reveal for now" nav toggle

A toggle in the nav that unblurs everything on screen for a while, for reading a whole list without tapping each item.

- Session-scoped (`sessionStorage` or in-memory). It must not change the saved `blur-text-<id>` setting.
- Clears on tab close, after a few idle minutes, and when the tab is hidden, like a single reveal.
- Likely implemented by making `applyBlur` also check a page-level "revealed all" flag, and re-running it over `[can-blur]`.

**Revisit if** tapping individual items starts to feel tedious, for example when you find yourself tapping through a whole list. Not needed until then.

## Ideas considered and rejected

- **Long-press:** collides with text selection and the context menu on mobile, and isn't discoverable.
- **Highlight to reveal:** no reliable hook; a selection can't be detected before text is selected, and it's awkward on touch.
- **Hover:** doesn't exist on phones and makes accidental reveals easy on desktop.

## Copy

The landing page, about page, and the "what is Contour Journal" blog post mention blur mode in general terms ("a blur mode if you're somewhere public"). They don't describe how reading works, so they didn't need to change. The blog sentence could take a clause like "and a tap on any blurred text shows it for a few seconds" if that's ever wanted.
