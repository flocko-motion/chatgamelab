.PHONY: rebase

# Force-pushes: only ever run this on your own feature branch.
rebase:
	@echo "Rebasing to origin/development..."
	git fetch origin development
	git rebase origin/development
	git push --force-with-lease
	@echo "✅ Rebased successfully."
