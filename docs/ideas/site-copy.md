# Ideas: the story the site tells

Status: ideas only, nothing changed. Written 2026-11-26 from a conversation and a read of the about page, landing page and three blog posts.

## The problem

- The copy in the blogs and marketing pages feels like a bad story, or maybe the wrong one.
- You want someone else to be able to use this and find it as useful as you have.
- Why it works for its builder may not carry over to someone else, so fit is hard to judge from the inside.
- It's unclear how anyone would find it among a thousand journal products, and whether the domain and Stripe are worth keeping.

## How we got here

It started with whether a sibling app should be "what this journal isn't" (a blog, or a commonplace book for ideas and quotes you do look back at). That led to whether the origin story, that other journals did things this one avoids, holds up against what exists. The search had not been exhaustive.

A quick skim of search results (mostly blogs by app makers, no apps tried) suggested:

- "No AI, no streaks, no gamification" is already advertised by others (Lite Journal, Pennen). Penzu and Diarium are described as quiet, plain journals.
- The loud end of the market does lean the way you felt (resurfacing, AI, mood stats).
- Nothing seen combined free writing, actions, thankfuls, routines that wait and optional patterns. That is not proof nothing does.
- On privacy you are not ahead of everyone. Encryption here is at rest with a key the server holds. Day One advertises end-to-end encryption.
- Some of these products may be newer than your decision to build.

## Where the current copy stands

**About** (`internal/web/landings/about.templ`). The strongest of the lot. "I built this for myself," "this is the version I actually kept using." Specific, and claims nothing about anyone else.

**Landing** (`internal/web/landings/landing.templ`). Reads like it was written to compete.

- The headline, "A journal that helps you notice patterns," describes a different product from "the point is to process and mostly leave it."
- "Why most journals don't last" and "nothing else felt right" lean on other tools falling short, which may not hold up.

**Why I Built Contour Journal** (`internal/blog/entries/why-i-built-contour-journal.md`). Holds the most personal material in any of the copy. Worth lifting out before removing anything. "None of them felt right" is the part to soften.

**What is Contour Journal?** (`internal/blog/entries/what-is-contour-journal.md`). Says "this one is special," and "patterns are a big focus here," which pulls the same way as the landing headline. Overlaps heavily with the about page.

**What's With the Name?** (`internal/blog/entries/whats-with-the-name.md`). Light and honest. Its idea that it's about the shape, not the details, is a gentler way to talk about patterns than the landing headline.

## Ideas explored

- **Tell it to one person.** The copy doesn't have to argue this is better. "I was going through a hard time, this is what I made, it helped me, you're welcome to use it."
- **Replace the existing posts.** You're considering removing them and writing new ones.
- **Blog only about this.** If you need to blog, this journal is basically the subject. No separate blog product.

## Where we landed

- The goal is knowing it helped someone. Income would be nice but needs numbers to amount to anything.
- Helping doesn't need SEO or standing out. It needs someone it might fit to hear about it and find it fits.
- People are more likely to arrive by a person mentioning it (you, a friend, a therapist, one honest post) than by searching.
- Keep the domain and Stripe. Drop the expectation that it has to grow. Paid stays what the about page calls it, a way to support the project.
- The about page's voice is the one to write the rest in.

## What we set aside

- **A separate blog product.** Plenty exist, and there was no distinct need.
- **A commonplace book app.** Not dropped, parked. It is a real counterpart (keep and return, versus write and leave), but the space is crowded. Try the habit with something plain first.
- **Competing on search or feature lists.** Very unlikely to work and not needed for the goal.

## Tensions and tradeoffs

- Wanting it to help others, while the reason it helps you partly can't be handed over.
- Help is mostly invisible by design. Someone could use it through a hard stretch and leave without a word, which looks the same as it not working.
- Telling a real person "this helped me" can feel more exposing than marketing.
- Even one stranger's private writing means owing them uptime, backups and a way out.
- Patterns are real and useful to you, so the copy can mention them. The question is whether they lead.

## Things that depend on the current copy

Check these if posts are removed or rewritten:

- `CLAUDE.md` and `README.md` point to the pages and blog entries as the source of truth for product philosophy. `CLAUDE.md` says the copy outranks its own summary, so removing posts before new ones exist leaves a thinner record.
- Pricing appears on the about page, in the "Cost" section of the "What is" post, and in the profile plan section. `CLAUDE.md` names all three.
- `docs/subscription-plan.md` references lines in the "What is" post.
- Existing links to the posts would presumably stop working. Not checked how the blog handles a removed post.
- Privacy wording should stay accurate in anything new: encrypted at rest, no ads, no tracking, no AI. Not end-to-end.

## Open questions

- Who is the one person you're writing to?
- What does the landing page lead with, if not patterns?
- Which lines from the old posts do you want to keep?
- Is "I use it pretty much every day" still how you'd put it?
- Remove old posts first, or only once replacements are ready?

## Rough size

Small in code (text in two templates and a few markdown files). The writing is the real work, and it's yours.
