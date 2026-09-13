# Contributing to ChatGameLab

This is the whole path a change takes, from your machine to the live site.
It is written for someone who has not done it here before, including web
designers who touch only the frontend.

## Before anything else

You need Git. Not deep knowledge — but branching, committing and pushing have
to be things you can do without stopping to think. On a Mac,
[GitHub Desktop](https://desktop.github.com/) or
[Sourcetree](https://www.sourcetreeapp.com/) give you all of it through a
window rather than a terminal.

You also need [`gh`](https://cli.github.com), the GitHub command line tool, for
the `make` commands below. `gh auth login` once and it remembers.

Setup, dependencies and how to run the thing locally are in
[README.md](README.md). This file is only about how a change travels.

## The two branches

**`development`** is where work lands. Every change goes here first, and the
development server redeploys itself from it automatically — so a merged change
is visible to everyone within a minute or two.

**`main`** is what the public uses. It only ever receives `development`, as a
whole, when the project lead decides the current state is ready. Nothing goes
to `main` directly, and no pull request should target it.

Your own work happens on a branch off `development`, named for what it does:
`fix/login-redirect`, `feat/workshop-export`.

## The path a change takes

### 1. Branch and work

Branch from an up-to-date `development`, and keep the change small. A branch
that lives for weeks is painful to review and painful to merge; one that lands
every few days is neither. If you are building something large, land it in
pieces that each make sense on their own.

While you work, `development` moves under you. Catch up with:

```sh
make rebase
```

which replays your commits on top of the latest `origin/development` and
force-pushes. Only ever run it on your own branch — it rewrites history, which
is fine on a branch nobody else has and destructive on one they do.

### 2. Open a pull request

```sh
make pr
```

From your feature branch this merges `origin/development` in — so the checks
run over the code that will actually land, rather than over your branch in
isolation — pushes, opens the pull request against `development`, and prints
its URL. Then it stops, leaving the pull request open for review.

Write a description that says what changed and why. A reviewer who has to read
the diff to find out what you were trying to do is a reviewer who will be slow.

### 3. The automated checks run

Opening the pull request starts them automatically. There are three:

| Check | What it does |
|---|---|
| **Backend Tests** | the Go test suite, minus the tests that call an AI provider |
| **Frontend Checks** | TypeScript type checking, then a production build |
| **Run Tests** | passes only if both of the above did |

They take a few minutes. A failure is not a judgement about you — it usually
means something in the repository moved and your branch has not caught up, or a
type is wrong somewhere you did not look. Push a fix to the same branch and
they run again. Nothing needs reopening.

Occasionally a check fails for a reason that has nothing to do with the change
— a network timeout pulling a container image, say. Re-run it from the pull
request page before going looking for a bug.

### 4. A human reviews it

Green checks mean the code compiles and the tests pass. They say nothing about
whether the change is a good idea, fits how the rest of the project works, or
does what its description claims. That is what review is for.

Expect comments, and expect some of them to ask for changes. Push new commits
to the same branch; the pull request updates itself and the checks run again.

### 5. Merge to `development`

Once it is approved:

```sh
make pr MERGE=1
```

waits until GitHub will let the pull request merge — which outlasts a required
check that has not started yet — and merges it. Or use the button on the pull
request page; they do the same thing.

**The development server deploys itself from this.** Merging to `development`
sends a signed trigger to the box, which builds the new code from source and
restarts. It takes a few minutes, and a timer catches anything the trigger
misses, so a merge is live within a quarter of an hour at worst. Nobody has to
deploy it.

Go and look at it. The development site is where a change is judged in
practice, and the gap between "the tests pass" and "it behaves as intended" is
where most of the remaining problems live.

### 6. Release to `main`

This step belongs to the project lead. When the state of `development` is
worth publishing:

```sh
make release
```

lists the commits that would enter `main`, says what merging them publishes,
and asks before opening anything. It then takes `development` to `main` through
a pull request, merges it, and merges `main` back into `development` so the
release tags stay reachable from where the work happens.

Merging to `main` publishes container images, tags the release, and deploys the
production instance. Version numbers come from the commit messages, so a commit
that says `fix:` and one that says `feat:` produce different releases — worth
writing accurately even though nothing enforces it.

## In short

```
your branch  →  pull request  →  checks  →  review  →  development  →  dev server (automatic)
                                                            ↓
                                                   release, by the lead
                                                            ↓
                                                     main  →  production (automatic)
```

## Things that save everyone time

**Small changes, often.** The single best thing you can do for review speed.

**Say what a change is for.** In the pull request description, and in the
commit message.

**Do not touch `main`.** Not a pull request against it, not a commit on it.
Releases are one operation performed by one person.

**Check the development site after your change lands.** It is the cheapest
place to find out you were wrong.

**Ask.** A question before you build something is much cheaper than a review
that concludes it was the wrong thing.
