# Idea: knowing whether anyone else uses it

Status: idea only, nothing built. Written 2026-11-26.

## The problem

- You'd like to know about potential impact, without anything that lets you read, or watch, people's journals.
- It also feels operational. If someone is relying on this, you don't want to do something risky with features or deploys, shut it down or change the domain without knowing.
- Usage is small enough that any count says something about particular people.

## Ideas explored

- **A user list** with subscription status and writing counts. Even without any text, it's a picture of someone's life next to their name. A burst of entries can mean a bad week.
- **A dashboard of totals.** Invites checking, which brings in the engagement metrics the product leaves out.
- **A query run by hand** before something risky.
- **One make target**, say `make usage`, so the query is decided once and you only ever see numbers.
- **Asking people instead.** A line near the feedback form, like "if this has helped, I'd love to hear it."

## Where we landed

One make target that prints one line:

- Accounts other than yours and the seed one that have written in the last 90 days.
- Accounts other than those that have ever written.

Shape of it: a read-only count query kept in the repo, run against production over the existing `ssh debian` access. No new route, page or stored data. No names, emails or text selected. Thankfuls and actions count as use, not just journal entries.

## What we set aside

- **A per-account breakdown.** Too personal, and not needed for the question.
- **A dashboard or admin screen.** See above.
- **More tools.** Hold at one. Add another only for a question you've needed to answer more than once.
- **Measuring impact.** Counts can't. Silence can mean it failed someone or that they're doing fine. A person saying so is the only real sign.

## Tensions and tradeoffs

- With few accounts, a count isn't anonymous in any real sense. If it says 1, you may know who.
- The privacy page (`internal/web/landings/privacy.templ`) says information is used "only to operate the journal." Add a plain sentence, for example: "I occasionally count totals across all accounts to see whether the journal is being used. I don't look at individual activity." Then keep it true. Looking at one account for billing or a reported bug is still operating the journal.
- A zero is not permission to be careless. Someone may come back much later, and someone who only reads or exports won't show in writing counts. Careful deploys and working backups hold while any account has writing.
- Before a shutdown or domain change, email every account with plenty of notice and a chance to export, whatever the count says.
- The count can fairly change how much ceremony something needs, such as whether downtime needs a heads-up.

## Open questions

- How to exclude your account and the seed one without putting an email address in the repo.
- Is 90 days the right window?
- Do deleted entries count toward "ever written"?
- Should routine-created action items be left out? They appear without the person doing anything, so counting them would overstate use.

## Rough size

Small. One query, one make target, one sentence on the privacy page.
