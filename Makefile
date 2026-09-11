.PHONY: rebase release sync check-clean-tree check-on-development

# Force-pushes: only ever run this on your own feature branch.
rebase:
	@echo "Rebasing to origin/development..."
	git fetch origin development
	git rebase origin/development
	git push --force-with-lease
	@echo "✅ Rebased successfully."

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

check-on-development:
	@test "$$(git rev-parse --abbrev-ref HEAD)" = development || { \
		echo "this runs on development, not $$(git rev-parse --abbrev-ref HEAD)"; \
		exit 1; }

# Resumable: a run whose merge landed and whose back-merge did not can be
# re-run, and picks up where it stopped rather than trying to open a pull
# request with nothing in it.
release: check-clean-tree check-on-development
	git fetch origin
	@git merge-base --is-ancestor origin/development HEAD \
		|| { echo "origin/development has commits you lack — pull first"; exit 1; }
	@if git merge-base --is-ancestor HEAD origin/main; then \
		echo ">> already merged into main — going straight to the back-merge"; \
	else \
		git push origin development; \
		echo ">> what this release takes to main:"; \
		git -c color.ui=never log --oneline --no-merges origin/main..HEAD | cat; \
		gh pr list --head development --base main --state open --json number \
			--jq '.[0].number' | grep -q . \
			|| gh pr create --base main --head development --fill; \
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
sync: check-on-development
	git fetch origin main
	@if git merge-base --is-ancestor origin/main HEAD; then \
		echo "✅ main is already an ancestor — nothing to merge back"; \
	else \
		git merge origin/main \
			-m "chore: merge main back, so release tags stay reachable" \
		&& git push origin development \
		&& echo "✅ merged main back — release tags are reachable from development"; \
	fi
