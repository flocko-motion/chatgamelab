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

## Things that save everyone time

**Small changes, often.** The single best thing you can do for review speed.

**Say what a change is for.** In the pull request description, and in the
commit message.

**Check the development site after your change lands.** It is the cheapest
place to find out you were wrong.

**Ask.** A question before you build something is much cheaper than a review
that concludes it was the wrong thing.

## The Process 

```
your branch  →  pull request  →  checks  →  review  →  development  →  dev server (automatic)
                                                            ↓
                                                   release, by the lead
                                                            ↓
                                                     main  →  production (automatic)
```


### Clone the Repo 

When you want to work on chatgamelab, you first need to clone the repo locally 
and create your personal branch in which you will do your changes. That is called a 
*feature branch*. 

### Creat a Feature Branch

When you work on changes, you do that on a branch that you create yourself. You 
branch it off from the `development` branch using the command: 
`git checkoug -b yourbranchname`

Branchnames should start with `fix/` for bugfixes, `feat/` for newly added features 
or `doc/` when working on documentation only. Those prefixes help others to 
quickly understand, what your branch is about. So e.g. `fix/broken-upload` would 
be the perfect branch name for fixing a broken upload function. 

Before you start working on your branch, run `make rebase` once. That syncs 
your banch against the current state of the project, so that you don't  
work on an old version. 

### Pull Request

Once your done you make a *pull request* to ask for your changes to be 
accepted into the project.

#### Creating a pull request 

You can either push your branch and create the pull request on github.com in the 
browser or you run this script in the repo: 

`make pr`

It runs a script which does everything that's needed. It pushes (uploads) 
your branch, creates the request to merge (Pull Request) and runs the automatic 
checks. 

Always make sure to provide an understandable description what you did. Don't flood
the description with AI generated details, nobody needs that. Write something short 
and on the point. 

#### The automated checks run

The automatic checks make sure, that your change doesn't break anything. 
Those tests are of course not perfect, they only test what they were programmed to test, 
that is running the typical use cases. 

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

#### Approval from Project Owner 

Once the tests pass, your PR (Pull Request) is visible on Github and the project 
owner (Florian Metzger-Noel) will get a notification that you want his review 
and want to merge your contribution. 

Once he confirmes, your changes are merged and are now in the *Development Branch*

### The Development Branch 

**`development`** is where your changework lands once our PR was accepted and merged. 
After merging it gets automatically deployed to the development server at 
*dev.cgl.fmnoel.de* where you can try out that version in a real environment. 

The development server exists, so that stakeholders can try out that version 
and assess, if it's ready for production. 

### The Main Branch - Releasing 

If the `development` branch seems to be good and the stakeholders signal, that it 
should be released, a PR (Pull Request) from `development` to `main` is created. This 
can be done with a browser in github.com or with the command 

`make release` 

Once merged, the updated *main* branch is automatically deployed to the production 
server. The new version is online within a vew minutes. You can tell from the 
changed version number on the website. 


