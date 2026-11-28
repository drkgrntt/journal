# Ideas: keeping good days visible

Status: ideas only, nothing built. Written 2026-11-26 from reading the code. Nothing here was run or tested on a device.

**Problem:** bad days get written about more than good days, so the record leans negative.

**Constraints:** privacy first, no streaks, no guilt, no AI reading entries.

## Conversation, 2026-11-26

This section is where things stand now. The earlier code reading is kept below it, and where the two disagree, this section wins.

### The problem

- Bad days get written about more than good days, so the record gives a bleaker picture than the life actually lived.
- Noticing good days is maintenance, like stretching. It won't stop a car accident or bad eating, but it keeps the mindset (and body) in the right place.
- A gap in journaling can mean things were mostly fine, so what gets written afterwards can skew negative. Not noticing the good is not necessarily why things go badly.
- Thankfuls exist for this, but they are easy to forget in the moment.
- Writing still matters. Getting thoughts out is a tangible way of acknowledging them, even if they are never read again.
- This is for anyone using the journal.

### Ideas explored

**A routine as the cue.** During a good stretch you aren't opening the app, so a better prompt inside it goes unseen. A routine with "email me" on is a cue you chose, and it waits if ignored. You can try this today with no changes. The routine email links to `/dashboard` (`internal/controllers/recurringActionItem.controller.go`), where the Today tab has the thankful box (`EntryBody` in `internal/web/journal/new.templ`). Writing there and ticking the item on the same screen is a small enough flow.

**Two suggested routines.** A thankful and a positive free write have different purposes: one names something good, the other stays with it for a while. They can go together but are separate. Draft wording:

- "Write down something you're thankful for"
- "Write about something good lately"

Tapping a suggestion would prefill the routine form (`internal/web/recurringActionItems/form.templ`). Nothing is created for anyone automatically. A weekly default seems gentler than daily.

**Move thankfuls above actions.** Today the whole action column, including every outstanding item, comes before thankfuls, so a long list pushes them out of view (desktop bottom right, off screen on mobile). `EntryBody` is shared by the new-entry page and the Today tab, so one change covers both. The journal form stays first. On mobile, thankfuls would likely still be below it, just much closer. Not checked in a rendered page.

**Reflecting on thankfuls.** A word cloud is the direction, not a requirement. What it should do:

- Big words (for example, the people someone loves) bring warmth at a glance.
- Small, rare words remind you of things you were once glad about and why.

Shapes considered:

| Shape | Good at | Weak at |
| --- | --- | --- |
| Word cloud, tap a word to see its whole lines | What recurs, plus the context behind it | Small words may be filler |
| Jar: a few whole thankfuls pulled at random, draw again | Rediscovery in your own sentences, works with only a handful | Shows recurrence only by chance |
| Dense, plain page of all lines by month | Noticing repeats yourself, no counting | No warmth at a glance, a bit generic |
| Ordinary list page (`ListPage` in `thankful/list.templ`, not routed) | Ownership: you can always get to them | Not a way of reflecting |

**Thankfuls in export.** Confirmed in code: `exportJournals` (`internal/controllers/profile.controller.go`) only includes thankfuls attached to an entry. Standalone ones are left out.

### Where we landed

- Fix the export so standalone thankfuls are included.
- Move thankfuls above actions.
- Offer the two suggested routines.
- A reflection view for thankfuls, with the word cloud as the leading candidate: size shows what keeps coming up, tapping a word shows a few of its lines. If real data makes the cloud mush, the jar works on its own.
- Thankful placement and reflection are free. Routines (the reminder) stay paid.
- That is also a sensible order. The cloud comes last, once there are thankfuls to put in it.

### What we set aside

- **Counts or rankings in the cloud** ("walk, 14"): that is the app interpreting you.
- **Merging the two prompts into one routine:** they do different jobs.
- **Checking off a routine without writing:** putting it into words is the point.
- **Entry templates as the cue:** they fill the journal text box, not the thankful box, are paid, and are only seen once you are already there.
- **Reflection over journal entries:** entries are a mixed bag. This is thankfuls only.
- **Web push and the tap-only check-in:** a cheaper input doesn't help someone who isn't thinking about it. Not ruled out for good.

### Tensions and tradeoffs

- The app's stance is that nothing analyzes or resurfaces what you write. A cloud counts words. The case for it: thankfuls only, your own words, on your own server, no AI, no conclusions, only when you go there, never emailed or pushed. That line should be drawn on purpose and said plainly in the product copy, so it doesn't spread to entries.
- The app is for being honest without feeling stupid about it. Having a list or view doesn't mean it's meant to be read, only that it can be. Knowing words are counted could still change how people write them.
- Noticing is valuable even if never reread. Reflection is a second, separate good, and shouldn't become the reason to write.
- A routine creates an action item, so you write in one place and tick it off in another.
- The practice shouldn't be asked to prevent bad stretches, or the next one feels like failing at maintenance.

### Open questions

- Will a routine actually work as the cue? Try one for a month.
- Do the small words mean anything on real data, or is it filler? How are common words and possessive forms of the same name handled?
- Where does the reflection view live: a page of its own, or a pattern on the Patterns tab (which people can hide)?
- Counting words means decrypting all of a user's thankfuls for each view. No pattern reads text today.
- Should the free thankful view be reachable for people who never use routines? (Assumed yes.)
- Which product copy changes: about page, blog entry, privacy policy?

### Rough size

| Piece | Size |
| --- | --- |
| Standalone thankfuls in export | Small |
| Move thankfuls above actions | Small |
| Suggested routines | Small |
| Thankful reflection view | Medium |

---

## Earlier code reading

## Where the ideas don't match the app today

- **There is no nudge notification.** The app has no web push at all (no subscriptions, no `push` handler in `sw.js`). The only nudge is the paid reminder email. "Reminders" in the app are your own notes, shown as a card on the Today tab and new-entry page.
- **Some charts already aggregate by day**, and some don't (table below).
- **Entry templates can't make a tap-only check-in.** A template is a name and a block of body text. It can't set a rating or offer chips.
- **A rating-only entry already mostly works.** Tap a rating, tap Done, leave the text empty.
- **A thankful that isn't attached to an entry is hard to see again** (details under idea 3).

## Summary

| Idea | Size | Suggested order |
| --- | --- | --- |
| 1. Chart by day | Small to medium | First. Check-ins would otherwise skew per-entry charts. |
| 3. Thankfuls as the prompt | Small | Second |
| 2. Cheap check-in | Medium (large with web push) | Third |

One limit applies to all three: charting by day stops a hard day counting several times, but it can't add the good days that were never logged. Only ideas 2 and 3 address that.

---

## 1. Chart by day, not by entry

### How it fits

Chart handlers are in `internal/controllers/dashboard*.go`, pure logic is in `internal/patterns/`, drawing is in `internal/web/assets/js/components/`.

| Pattern | Counts today | Change needed |
| --- | --- | --- |
| Calendar (rating) | Day average (computed in the browser) | None |
| Year in pixels | Day average (`patterns.DailyRatings`) | None |
| Rating flow | Day average | **Gaps are drawn over** (see below) |
| Rating vs thankfulness | Days | None |
| Rating vs action completion | Days | None |
| How you tend to feel | **Entries** | Count days by rounded average |
| Rating by topic | **Entries** | One value per topic per day, then average |
| Week by time of day | **Entries** | One value per day per time block |
| What you're writing about | **Entries** | Count days per topic |
| Calendar (wrote / thankful / did) | Items per day, on purpose | Probably none |
| Routines you've done | Completions, not ratings | None |

**Gaps:** `getMoodChart` already sends `null` for empty days, but `moodChart.js` sets `spanGaps: true`, which draws the line straight across them. No chart turns an empty day into zero.

### What would change

- **Data model:** none.
- **Queries:** no new ones. `days` windows and filters stay the same.
- **Logic:** reuse or extend `patterns.DailyRatings`. Update `ComputeWeekByTime` and `ComputeRatingDistribution` inputs, `getMoodByTopic` and `getTopicDistribution`, with tests in `internal/patterns/`.
- **UI:** set `spanGaps: false`. Change "entries" wording in tooltips (`ratingDistribution.js`, `weekByTime.js`, `yearInPixels.js`, `ratingCalendar.js`) and in the descriptions in `dashboardPage.templ`.
- **Encryption:** no impact if handlers select only date, rating and topic.

### Open questions and tradeoffs

- **Is a mean the right day value?** An awful morning and a great evening become "fine". The alternative (last rating of the day) hides more. Recommendation: mean.
- **Rounding for distribution:** a 3.5 day has to land somewhere. The comparison charts already round half up (`bucketDaysByRating`). Reuse that.
- **With gaps on, sparse weeks become isolated dots.** That is honest, but the line chart will look broken up.
- **Week by time:** the "faint under 3" rule would mean 3 days. More squares will be faint.
- **Topic charts:** a day with two topics counts once in each. Day counts won't add up to total days.

### Noticed while reading (separate from this idea)

- **Privacy:** the rating calendar sends whole journal records to the page as JSON, including decrypted entry text (`getCalendar`, `calendar-journals` in `ratingCalendar.templ`). It only needs date and rating. It goes only to the signed-in owner, but it is more exposure than needed and worth fixing on its own.
- `getMoodChart`, `getMoodByTopic`, `getTopicDistribution` and both comparison handlers also load (and decrypt) entry text they don't use.
- The default Rating flow view ends at the start of today, so today's entries are missing.

**Size:** small if limited to the four per-entry charts and the gap setting. Medium with the cleanup above.

---

## 2. Cheap good-day logging

### How it fits

- A check-in can be an ordinary `Journal` (`internal/models/journal.model.go`): date, rating, topic, empty or one-line `Entry`. It then shows up in every pattern, the journal list and export with no extra work.
- Rating buttons already exist (`internal/web/journal/form.templ`, `ratingPicker.js`). Date defaults to now and topic to General.
- Offline support already exists for entries (`autosave.js`, `utils/drafts.js`) and thankfuls (`utils/outbox.js`).

### Template system or separate?

**Recommendation: separate, small view.** The template system (`entryTemplate.model.go`, `templatePicker.js`) is the wrong fit:

- Text only, no rating or chips.
- Paid. A basic way to log a good day arguably belongs in the free core.
- Fetched at runtime and unavailable offline, by design.
- Picking one fills the textarea and focuses it, which opens the keyboard.

### What would change

- **UI:** a new Templ view with rating buttons, chips, one optional line and Save. Either its own route (such as `/check-in`) or a compact mode on the Today tab.
- **Handler:** can post to the existing `POST /api/journal`, including `draftId`. A new view route needs `RequireAuth`.
- **Offline:** a new route must be added to `sw.js` (`warm`, `SHELL_PATHS`).
- **Chips:** the open design choice.

| Where chips are stored | Data model | Encryption | Notes |
| --- | --- | --- | --- |
| Joined into `Entry` text | None | Already encrypted | Simplest. Recommended to start. |
| Each chip becomes a thankful | None | Already encrypted | Only fits "good thing" chips |
| New table of your own chips | Migration, management screen | Names must be encrypted | Most work |
| `Journal.Metadata` | None | **Plaintext jsonb** | Avoid for anything personal |

### Offline rating-only entries

A rating-only entry submitted offline used to be dropped. Fixed in `autosave.js` after this doc was written (tests in `autosave.test.js`, not checked in a real browser): pressing Done now keeps the draft even with no text. A check-in view that doesn't reuse `autosave.js` needs the same behavior.

### Deep link from the nudge

- **Email (what exists):** the link in `internal/emails/reminder.templ` goes to `/dashboard`. Pointing it at the check-in is a small change. It could be a setting.
- **Signed out:** `RequireAuth` redirects to login without remembering where you were going.
- **Native shell:** `mobile/` declares no app links, so an email link should open in the browser, not the app.

### Web push, if wanted later

| | Android (Chrome / installed PWA) | iOS / iPadOS |
| --- | --- | --- |
| Web push | Yes | 16.4+, only for sites added to the Home Screen |
| Permission | User prompt | User prompt, after a tap, in the Home Screen app |
| Action buttons on the notification | Yes | Not supported (caniuse, through 27.2) |
| Type a reply in the notification | Don't count on it (unconfirmed) | No action support to build on |
| Tap opens a chosen page | Yes (standard behavior) | Expected, test on a device |
| Inside our Capacitor shell | Android WebView has no Push API | Would likely need native push too (unconfirmed) |

- **Design for tapping the notification to open the check-in.** Action buttons ("Good", "Great") would be an Android-only extra. The number shown is limited (commonly two, not confirmed here), so five ratings won't fit.
- **Logging straight from a button** means the service worker writes with your session cookie, and needs its own offline and expired-session handling.
- **Needs:** a subscriptions table (migration), VAPID keys, a `push` and `notificationclick` handler in `sw.js`, a sender job beside `jobs/reminderEmail.job.go`, and likely a new Go dependency.
- **Privacy:** notifications go through Google's or Apple's push service. Payloads are encrypted in transit by the protocol, but keep them generic anyway, with no user text. Subscription endpoints are device identifiers and the privacy policy (`landings/privacy.templ`) would need updating.
- **Native push** would add a third party SDK (such as FCM) to the shell. That is a bigger privacy decision.

Sources: [caniuse: notification actions](https://caniuse.com/mdn-api_notification_actions), [MDN: showNotification](https://developer.mozilla.org/en-US/docs/Web/API/ServiceWorkerRegistration/showNotification), [iOS web push requirements](https://progressivewebapps.com/background/ios-web-push/), [Android WebView push](https://code2native.com/blog/webview-push-notifications-android). The last two are third party write-ups. Verify on real devices before building.

### Open questions and tradeoffs

- Should check-ins look different from written entries in the journal list? An empty card may read as "unfinished".
- Do they count in the "wrote" lens and "What you're writing about"? Without idea 1 and some choice here, they inflate General.
- Fixed chips or your own? Fixed is cheaper but may feel generic.
- Free or paid? Recommendation: free check-in. Any push or email nudge follows the current paid reminder.
- A very easy check-in might replace writing on days you would have written. Only use will tell.

**Size:** medium (view, chips as entry text, offline support, email link). Web push is a separate large piece.

---

## 3. Thankfuls as the good-day prompt

### How it fits

- `Thankful` (`internal/models/thankfuls.model.go`) is already one encrypted line, with an optional `JournalID`.
- The form (`internal/web/thankful/form.templ`) is already on the Today tab and new-entry page (`journal/new.templ`), after the journal form.
- It already works offline through the outbox.
- The calendar's "thankful" lens already shows days that have one.

### What's different from what you might expect

- A thankful added before an entry exists is attached if that entry is then saved. If not, it stays standalone.
- **Standalone thankfuls show only as a count in the calendar lens.** No routed page lists them (`ListPage` in `thankful/list.templ` exists, but no route renders it).
- **They are not in export.** `exportJournals` only includes thankfuls attached to entries. This looks like a gap worth fixing whatever happens with this idea.
- **They don't affect any rating chart.** A thankful has no rating, and "Rating vs thankfulness" only looks at ones attached to rated entries.
- **Offline dates:** the outbox doesn't send when a thankful was written, so it is dated when it syncs.

### What would change

- **UI:** a short prompt such as "One thing that went right?" with the existing input, placed above the journal form or on its own small view. Wording in `thankful/form.templ` and the heading in `journal/new.templ`.
- **Data model:** none for the minimum version. Marking prompt answers separately would use `Thankful.Metadata` (a flag is fine in plaintext, text is not).
- **Queries:** none for the prompt. Viewing standalone thankfuls needs a route with `RequireAuth`, scoped to the user and paginated (`getThankfuls` already does the last two).
- **Export:** add standalone thankfuls.
- **Offline:** send a created time, clamped on the server as `completedAt` is for action items.
- **Encryption:** no change. `Thankful.Text` is already encrypted at rest, and the prompt itself adds no user text to the cached new-entry page. Offline thankfuls sit in plaintext in the on-device outbox until they sync, as they do now.

### Open questions and tradeoffs

- Replace "Today I'm Thankful For", or add a second prompt? Recommendation: reword the existing one rather than have two.
- Prominence versus pressure: above the entry is easier to find but changes what the page asks first.
- Is "visible in the thankful lens" enough? If good days should show in rating charts, the prompt needs an optional rating, and then it is really idea 2.
- Should standalone thankfuls be browsable? That helps export and ownership, but a browsable "good things" list is a step toward resurfacing. Keep it something you go to, never something pushed at you.

**Size:** small for the prompt alone. Small to medium with export, offline date and a list page.

---

## How 2 and 3 could combine

One small screen: rating buttons, then "one thing that went right" (optional, saved as a thankful on that check-in). Chips can wait. This reuses encrypted storage that exists, needs no migration, and tests whether the cheap path gets used before adding chips or push.
