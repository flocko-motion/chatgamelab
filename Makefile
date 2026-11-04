.PHONY: help rebase pr release sync check-clean-tree check-on-feature

# The release line this copy of the file belongs to. Each major version keeps
# its own copy, so these two lines are what differs from one line to the next:
# 1.X uses development and main, 2.X uses v2/development and v2/main.
DEV  := v2/development
MAIN := v2/main

# First target, so a bare `make` prints this instead of force-pushing.
help:
	@echo "make rebase           rebase onto origin/$(DEV) and force-push (own feature branch only)"
	@echo "make pr [MERGE=1]     merge $(DEV) in, push, open the pull request; MERGE=1 waits for checks and merges"
	@echo "make release [YES=1]  $(DEV) -> $(MAIN), then $(MAIN) back into $(DEV); YES=1 skips the prompt"
	@echo "make sync             merge $(MAIN) back into $(DEV) through a pull request"

# Force-pushes: only ever run this on your own feature branch.
rebase:
	@echo "Rebasing to origin/$(DEV)..."
	git fetch origin $(DEV)
	git rebase origin/$(DEV)
	git push --force-with-lease
	@echo "✅ Rebased successfully."

# Whether a pull request may merge is GitHub's question, and mergeStateStatus
# answers it in one word for two situations: BLOCKED is both "a required check
# has yet to report" and "the base branch policy refuses this merge", and only
# the first clears by waiting. So the wait is on the required checks named by
# the base branch's own rules, which says immediately when there is nothing
# left to wait for. `gh pr checks` is watched for the eye and trusted for
# nothing: it reports what sits on the head commit, which for the back-merge is
# main's own release build, green before the pull request existed.
#
# A base branch can still refuse the merge for a rule no amount of waiting will
# satisfy — one that restricts who may update the ref, say. gh names the rule,
# and that is where the run stops.
define wait_and_merge
passed_checks() { \
	gh pr view "$$1" --json statusCheckRollup \
		--jq '.statusCheckRollup[]? | select((.conclusion // .state) == "SUCCESS") | .name // .context' \
		2>/dev/null; \
}; \
broken_checks() { \
	gh pr view "$$1" --json statusCheckRollup \
		--jq '.statusCheckRollup[]? | select(.conclusion == "FAILURE" or .conclusion == "ERROR" or .conclusion == "TIMED_OUT" or .conclusion == "CANCELLED" or .conclusion == "STARTUP_FAILURE" or .state == "FAILURE" or .state == "ERROR") | .name // .context' \
		2>/dev/null; \
}; \
required_checks() { \
	gh api "repos/{owner}/{repo}/rules/branches/$$1" \
		--jq '.[] | select(.type == "required_status_checks") | .parameters.required_status_checks[].context' \
		2>/dev/null | sort -u; \
}; \
unmet_checks() { \
	passed="$$(passed_checks "$$1")"; \
	required_checks "$$2" | while read -r ctx; do \
		[ -n "$$ctx" ] || continue; \
		printf '%s\n' "$$passed" | grep -qxF "$$ctx" || printf '%s\n' "$$ctx"; \
	done; \
}; \
wait_and_merge() { \
	pr="$$1"; \
	base="$$(gh pr view "$$pr" --json baseRefName --jq .baseRefName)"; \
	echo ">> watching #$$pr's checks…"; \
	: "a check run takes a moment to register on a freshly pushed head, and"; \
	: "--watch returns at once while none has"; \
	for i in $$(seq 1 12); do \
		n="$$(gh pr view "$$pr" --json statusCheckRollup \
			--jq '.statusCheckRollup | length' 2>/dev/null)"; \
		[ "$${n:-0}" -gt 0 ] && break; \
		sleep 5; \
	done; \
	gh pr checks "$$pr" --watch --fail-fast || true; \
	echo ">> waiting for #$$pr's required checks to pass…"; \
	for i in $$(seq 1 180); do \
		broken="$$(broken_checks "$$pr")"; \
		if [ -n "$$broken" ]; then \
			echo "#$$pr has checks that did not pass:"; \
			printf '%s\n' "$$broken" | sed 's/^/     /'; \
			return 1; \
		fi; \
		if [ "$$(gh pr view "$$pr" --json mergeStateStatus \
			--jq .mergeStateStatus)" = DIRTY ]; then \
			echo "#$$pr has conflicts — resolve them, then re-run"; \
			return 1; \
		fi; \
		unmet="$$(unmet_checks "$$pr" "$$base")"; \
		[ -z "$$unmet" ] && break; \
		sleep 10; \
	done; \
	if [ -n "$$unmet" ]; then \
		echo "#$$pr is still waiting on these required checks after 30 minutes:"; \
		printf '%s\n' "$$unmet" | sed 's/^/     /'; \
		return 1; \
	fi; \
	echo ">> merging #$$pr…"; \
	gh pr merge "$$pr" --merge --delete-branch=false \
		|| { echo "$$base refused #$$pr — the message above names the rule"; \
		     return 1; }; \
}
endef

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
	case "$$branch" in $(DEV)|$(MAIN)|HEAD) \
		echo "this runs on a feature branch, not $$branch"; exit 1;; esac

# Resumable: a branch whose pull request is already merged says so and stops,
# rather than opening a second one.
pr: check-clean-tree check-on-feature
	git fetch origin $(DEV)
	@branch="$$(git rev-parse --abbrev-ref HEAD)"; \
	if git merge-base --is-ancestor HEAD origin/$(DEV); then \
		echo "✅ $$branch is already in $(DEV) — nothing to open"; \
		exit 0; \
	fi; \
	if git merge-base --is-ancestor origin/$(DEV) HEAD; then \
		echo ">> origin/$(DEV) is already an ancestor"; \
	else \
		echo ">> merging origin/$(DEV) in…"; \
		git merge origin/$(DEV) -m "chore: merge $(DEV) into $$branch" \
			|| { echo "resolve the conflicts, commit, then re-run"; exit 1; }; \
	fi; \
	git push --set-upstream origin "$$branch"; \
	echo ">> what this pull request takes to $(DEV):"; \
	git -c color.ui=never log --oneline --no-merges origin/$(DEV)..HEAD | cat; \
	gh pr list --head "$$branch" --base $(DEV) --state open --json number \
		--jq '.[0].number' | grep -q . \
		|| gh pr create --base $(DEV) --head "$$branch" --fill \
		|| { echo "could not open the pull request"; exit 1; }; \
	if [ -z "$(MERGE)" ]; then \
		echo "✅ open for review: $$(gh pr view "$$branch" --json url --jq .url)"; \
		echo "   to wait for its checks and merge it:  make pr MERGE=1"; \
		exit 0; \
	fi; \
	$(wait_and_merge); \
	wait_and_merge "$$branch" || exit 1; \
	echo "✅ merged into $(DEV) — 'make release' puts it on $(MAIN)"

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
	@if git rev-parse --verify --quiet refs/heads/$(DEV) >/dev/null \
		&& ! git merge-base --is-ancestor $(DEV) origin/$(DEV); then \
		echo "your local $(DEV) has commits origin lacks — push them first"; \
		exit 1; \
	fi
	@if git merge-base --is-ancestor origin/$(DEV) origin/$(MAIN); then \
		echo ">> already merged into $(MAIN) — going straight to the back-merge"; \
	else \
		notes="$$(git -c color.ui=never log --pretty='%h %s' --no-merges \
			origin/$(MAIN)..origin/$(DEV))"; \
		echo ">> these commits enter $(MAIN):"; \
		echo "$$notes" | sed 's/^/     /'; \
		echo ">> the 2.X line publishes nothing yet: no release tag, no images, no"; \
		echo "   deploy. Merging only moves $(MAIN)."; \
		if [ -z "$(YES)" ]; then \
			printf ">> really release to $(MAIN)? [y/N] "; \
			read -r reply < /dev/tty || reply=""; \
			case "$$reply" in \
				[yY]|[yY][eE][sS]) ;; \
				*) echo "aborted"; exit 1;; \
			esac; \
		fi; \
		: "--fill reads the local $(DEV) branch, which this target never"; \
		: "checks out or updates, so the range it sees can be empty"; \
		gh pr list --head $(DEV) --base $(MAIN) --state open --json number \
			--jq '.[0].number' | grep -q . \
			|| gh pr create --base $(MAIN) --head $(DEV) \
				--title "release $$(date +%Y-%m-%d)" --body "$$notes" \
			|| { echo "could not open the pull request"; exit 1; }; \
		$(wait_and_merge); \
		wait_and_merge $(DEV) || exit 1; \
	fi
	@$(MAKE) --no-print-directory sync
	@echo "✅ released — $(MAIN) is up to date"

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
	git fetch origin $(MAIN) $(DEV)
	@if git merge-base --is-ancestor origin/$(MAIN) origin/$(DEV); then \
		echo "✅ $(MAIN) is already an ancestor — nothing to merge back"; \
		exit 0; \
	fi; \
	gh pr list --head $(MAIN) --base $(DEV) --state open --json number \
		--jq '.[0].number' | grep -q . \
		|| gh pr create --base $(DEV) --head $(MAIN) \
			--title "chore: merge $(MAIN) back, so release tags stay reachable" \
			--body "Carries no file changes. Puts the release tags on $(MAIN) where git describe on $(DEV) can reach them." \
		|| { echo "could not open the back-merge pull request"; exit 1; }; \
	num="$$(gh pr list --head $(MAIN) --base $(DEV) --state open --json number \
		--jq '.[0].number')"; \
	: "--delete-branch=false, which wait_and_merge passes, matters more here"; \
	: "than anywhere else: the head of this pull request is $(MAIN)"; \
	$(wait_and_merge); \
	wait_and_merge "$$num" || exit 1; \
	echo "✅ merged $(MAIN) back — release tags are reachable from $(DEV)"; \
	: "git refuses this one while $(DEV) is the branch you have out, which"; \
	: "is the case the hint below is for"; \
	git fetch --quiet origin $(DEV) $(DEV):$(DEV) 2>/dev/null \
		|| git fetch --quiet origin $(DEV); \
	if git rev-parse --verify --quiet refs/heads/$(DEV) >/dev/null \
		&& ! git merge-base --is-ancestor origin/$(DEV) $(DEV); then \
		echo "   your $(DEV) checkout is behind it — git pull to catch up"; \
	fi
