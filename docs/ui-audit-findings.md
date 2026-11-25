# UI/UX audit findings

Audit date: 2026-11-25. Method: headless Chromium (Playwright) against the local dev server, logged in as the seeded `test@example.com` user (~400 days of data, purple palette), at three viewports in light and dark:

| Name | Size |
|---|---|
| mobile | 375x812 (touch) |
| tablet | 768x1024 |
| desktop | 1440x900 |

Pages covered: landing, about, blog, privacy, feedback, login, register, forgot, Today, Patterns, journal list / entry / edit / new, action items, entry templates, profile, custom topics, reminders. All returned 200. Screenshots were not committed.

This is a visual pass only. Nothing here is a bug fix yet, and the findings are for a later review. Several items are judgment calls against the product philosophy in `CLAUDE.md`.

## Worth a look

1. **Calendar draws empty days as heavy black dots (Patterns, rating lens).** The legend even lists "No Journals" as a category. `CLAUDE.md` says empty days should be plain blank, with no "missed" styling. After the last entry, a whole run of black dots reads like a gap to be filled. Candidate: draw empty days as a faint outline or nothing, and drop "No Journals" from the legend.
2. **Rating buttons wrap awkwardly.** On the new-entry form the five pills break 3+2 (centered) on mobile, and 2+2+1 inside the narrow left column at tablet width, leaving "Awful" alone. Candidates: a single row that scales, or a fixed 5-up grid.
3. **Tablet (768 px) Today / New Entry is cramped.** The two-column layout leaves the Take Action column narrow, so action text wraps to 3-4 lines. In dark mode the thankful input placeholder is clipped ("What are you thar...") and the "Today I'm Thankful For" heading wraps to two lines. Candidate: switch to one column below about 900 px, or give the action column a minimum width.
4. **Desktop Today is lopsided.** The entry form is short and the action list is very long (28 outstanding in the seed data), so the left column is empty for most of the page. The seed data exaggerates this, but a heavy real-world routine list would do the same. Candidate: cap the list height with an internal scroll, or collapse after N items.
5. **Icon-only buttons.** The edit pencil, shuffle, bookmark, mic and the Patterns "hide" eye have no visible label. I could not check `aria-label` or tooltips from screenshots, so confirm in the DOM.

## Minor

- **Entry Templates list:** the template body renders with large uneven gaps between lines (the "End of day" example), while "Brain dump" is tight. It looks like whitespace preserved from the stored text rather than a deliberate design.
- **Login/register use the default (teal) palette while the signed-in app uses the user's palette (purple here).** That's expected, since the palette is paid and only applies when signed in. Mentioned only because the jump is noticeable on logout.
- **Console errors during the run:** a stylesheet request answered with HTML (MIME error) and a 403 fetching the service worker script. This is most likely the bot filter reacting to headless Chromium's client hints, not a real bug, but it is unconfirmed. Re-check in a real browser's DevTools.

## Looked fine

- The mobile tab bar clears page content at the bottom of Today (the last input sits fully above the bar), and journal lists and entries scroll cleanly under it.
- Landing, profile, journal list, entry view, and logged-out auth pages are clean and readable on mobile and desktop, in light and dark.
- Dark mode contrast is good on every page reviewed.
- No horizontal overflow was flagged on the pages whose results I saw (the first pass checked `scrollWidth > innerWidth`). The first mobile-light results scrolled out of view, so treat that check as incomplete.
- Patterns charts render on mobile. The year grid is small but legible, and the week-by-time heatmap looks good.

## Not verified

- **The slide-out (hamburger) menu.** The script's selector did not find the button, so no menu screenshot exists.
- **Real-device behavior:** PWA install, Capacitor safe areas, service-worker offline path, touch targets by hand.
- **Interactions:** only static renders. Nothing was clicked beyond login (no form errors, HTMX swaps, empty states, or the blur mode).
- **Empty / new-user state:** the seed user has a full year of data, so first-run screens were not seen.
- **Light-mode tablet and desktop shots for the pages not named above** were captured but only spot-checked.

## Re-running

- The script is a throwaway Playwright file (not in the repo): log in once, reuse the saved session, then visit each page per viewport and theme.
- Gotchas:
  - Headless Chromium needs a normal browser User-Agent, or the bot filter returns 403.
  - Login is limited to 10 per 15 minutes per client. Log in once and reuse the session. On a local dev server, a different `X-Forwarded-For` header per run sidesteps the limiter without touching code or restarting the server.
  - Full-page screenshots put fixed elements (the tab bar) mid-page, so use viewport-only shots to judge them.
