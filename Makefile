.PHONY: rebase pr release sync check-clean-tree check-on-feature

# Force-pushes: only ever run this on your own feature branch.
rebase:
	@echo "Rebasing to origin/development..."
	git fetch origin development
	git rebase origin/development
	git push --force-with-lease
	@echo "✅ Rebased successfully."

# ── pr ───────────────────────────────────────────────────────────────────────
#
# feature branch -> development: merge development in, push, open the pull
# request, and stop there, which leaves room for a review. `make pr MERGE=1`
# carries on and waits for the checks and merges. Getting development onto main
# is `make release`.
#
# The first step is `sync` pointed the other way. origin/development is made an
# ancestor before the pull request opens, so its checks run over the code that
# will actually land rather than over a base the branch has drifted from. A
# feature branch can be rewritten, so `make rebase` reaches the same place more
# cheaply; the merge is what a run that must not touch anyone's checkout can do
# on its own.

check-on-feature:
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	case "$$branch" in development|main|HEAD) \
		echo "this runs on a feature branch, not $$branch"; exit 1;; esac

# Resumable: a branch whose pull request is already merged says so and stops,
# rather than opening a second one.
pr: check-clean-tree check-on-feature
	git fetch origin development
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	if git merge-base --is-ancestor HEAD origin/development; then \
		echo "✅ $$branch is already in development — nothing to open"; \
		exit 0; \
	fi; \
	if git merge-base --is-ancestor origin/development HEAD; then \
		echo ">> origin/development is already an ancestor"; \
	else \
		echo ">> merging origin/development in…"; \
		git merge origin/development -m "chore: merge development into $$branch" \
			|| { echo "resolve the conflicts, commit, then re-run"; exit 1; }; \
	fi; \
	git push --set-upstream origin "$$branch"; \
	echo ">> what this pull request takes to development:"; \
	git -c color.ui=never log --oneline --no-merges origin/development..HEAD | cat; \
	gh pr list --head "$$branch" --base development --state open --json number \
		--jq '.[0].number' | grep -q . \
		|| gh pr create --base development --head "$$branch" --fill \
		|| { echo "could not open the pull request"; exit 1; }; \
	if [ -z "$(MERGE)" ]; then \
		echo "✅ open for review: $$(gh pr view "$$branch" --json url --jq .url)"; \
		echo "   to wait for its checks and merge it:  make pr MERGE=1"; \
		exit 0; \
	fi; \
	echo ">> waiting for the pull request's checks…"; \
	: "gh reports no checks both where a base requires none and in the"; \
	: "seconds before checks register, so probe before watching"; \
	for i in 1 2 3 4 5; do \
		gh pr checks "$$branch" >/dev/null 2>&1 && break; \
		sleep 3; \
	done; \
	gh pr checks "$$branch" --watch --fail-fast \
		|| { echo "a check is failing — fix it, then re-run"; exit 1; }; \
	echo ">> merging…"; \
	gh pr merge "$$branch" --merge --delete-branch=false; \
	echo "✅ merged into development — 'make release' puts it on main"

# ── release ──────────────────────────────────────────────────────────────────
#
# development -> main, and then main back into development.
#
# That second half is the point, and it is what keeps release tags findable.
# Every release is a "Merge pull request from development" commit created on
# main, and semantic-release tags that commit — which development can never
# reach on its own, being its parent rather than its descendant. Left alone the
# two branches diverge a little at every release, and anything asking git which
# release a commit belongs to walks back past all of them to the last commit
# they happened to share: `git describe` on development reported v1.47.0 while
# main was at v1.56.0, and the gap only ever grows.
#
# Merging main back closes it. The merge carries no files — those merge commits
# have empty diffs against their development parent — so it changes the history
# graph and nothing in the tree.
#
# A repo with only short-lived branches rebases them onto the default branch
# after the merge instead, which is cheaper. development is shared, so
# rewriting it would break everyone else's clone; a merge is the version of
# that step available to a branch you cannot rewrite.

check-clean-tree:
	@git diff --quiet && git diff --cached --quiet \
		|| { echo "working tree is dirty — commit or stash first"; exit 1; }

# Resumable: a run whose merge landed and whose back-merge did not can be
# re-run, and picks up where it stopped rather than trying to open a pull
# request with nothing in it.
#
# Every ref here is an origin/ ref, so this runs from whatever branch you have
# out, mid-edit, and leaves your tree alone. The one thing a checkout of
# development would lend it is somewhere to make the back-merge commit, and
# `sync` brings its own.
#
# It lists what would enter main and asks before opening anything. `make
# release YES=1` answers that prompt, for a run with no terminal to ask at.
release:
	git fetch origin
	@if git rev-parse --verify --quiet refs/heads/development >/dev/null \
		&& ! git merge-base --is-ancestor development origin/development; then \
		echo "your local development has commits origin lacks — push them first"; \
		exit 1; \
	fi
	@if git merge-base --is-ancestor origin/development origin/main; then \
		echo ">> already merged into main — going straight to the back-merge"; \
	else \
		notes="$$(git -c color.ui=never log --pretty='%h %s' --no-merges \
			origin/main..origin/development)"; \
		echo ">> these commits enter main:"; \
		echo "$$notes" | sed 's/^/     /'; \
		echo ">> merging them publishes: semantic-release tags main and cuts a"; \
		echo "   GitHub release, and CI pushes the images production deploys."; \
		if [ -z "$(YES)" ]; then \
			printf ">> really release to main? [y/N] "; \
			read -r reply < /dev/tty || reply=""; \
			case "$$reply" in \
				[yY]|[yY][eE][sS]) ;; \
				*) echo "aborted"; exit 1;; \
			esac; \
		fi; \
		: "--fill reads the local development branch, which this target never"; \
		: "checks out or updates, so the range it sees can be empty"; \
		gh pr list --head development --base main --state open --json number \
			--jq '.[0].number' | grep -q . \
			|| gh pr create --base main --head development \
				--title "release $$(date +%Y-%m-%d)" --body "$$notes" \
			|| { echo "could not open the pull request"; exit 1; }; \
		echo ">> waiting for the pull request's checks…"; \
		: "gh reports no checks reported both where a base requires none and"; \
		: "in the seconds before checks register, so probe before watching"; \
		for i in 1 2 3 4 5; do \
			gh pr checks development >/dev/null 2>&1 && break; \
			sleep 3; \
		done; \
		gh pr checks development --watch --fail-fast \
			|| { echo "a check is failing — fix it, then re-run"; exit 1; }; \
		echo ">> merging…"; \
		gh pr merge development --merge --delete-branch=false; \
	fi
	@$(MAKE) --no-print-directory sync
	@echo "✅ released — CI tags main and builds both branches"

# The back-merge on its own: for a release whose pull request someone else
# merged, or which was merged in the browser. Safe to run at any time, and a
# no-op when main is already an ancestor.
#
# development takes no direct push — a repository ruleset requires a pull
# request and passing checks — so the back-merge arrives as one too, opened
# from main into development. GitHub makes the merge commit, which is why
# nothing here is checked out or merged locally and why this runs from any
# branch, mid-edit, without touching the tree.
sync:
	git fetch origin main development
	@if git merge-base --is-ancestor origin/main origin/development; then \
		echo "✅ main is already an ancestor — nothing to merge back"; \
		exit 0; \
	fi; \
	gh pr list --head main --base development --state open --json number \
		--jq '.[0].number' | grep -q . \
		|| gh pr create --base development --head main \
			--title "chore: merge main back, so release tags stay reachable" \
			--body "Carries no file changes. Puts the release tags on main where git describe on development can reach them." \
		|| { echo "could not open the back-merge pull request"; exit 1; }; \
	num="$$(gh pr list --head main --base development --state open --json number \
		--jq '.[0].number')"; \
	echo ">> waiting for the back-merge pull request's checks…"; \
	: "gh reports no checks both where a base requires none and in the"; \
	: "seconds before checks register, so probe before watching"; \
	for i in 1 2 3 4 5; do \
		gh pr checks "$$num" >/dev/null 2>&1 && break; \
		sleep 3; \
	done; \
	gh pr checks "$$num" --watch --fail-fast \
		|| { echo "checks did not pass on #$$num — fix them, then re-run"; exit 1; }; \
	: "--delete-branch=false matters more here than anywhere else: the head of"; \
	: "this pull request is main"; \
	gh pr merge "$$num" --merge --delete-branch=false \
		|| { echo "could not merge #$$num — merge it in the browser"; exit 1; }; \
	echo "✅ merged main back — release tags are reachable from development"; \
	: "git refuses this one while development is the branch you have out, which"; \
	: "is the case the hint below is for"; \
	git fetch --quiet origin development development:development 2>/dev/null \
		|| git fetch --quiet origin development; \
	if git rev-parse --verify --quiet refs/heads/development >/dev/null \
		&& ! git merge-base --is-ancestor origin/development development; then \
		echo "   your development checkout is behind it — git pull to catch up"; \
	fi
