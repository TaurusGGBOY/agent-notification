## Repository Workflow

- Keep the repository root worktree on `main` long-term; use it only for syncing, inspection, and worktree management.
- Do not modify project files in the repository root worktree for normal tasks.
- For every code or documentation change, create one feature branch per task in its own worktree.
- Do development and verification inside the feature worktree only.
- Before committing, run the relevant automated tests and decide whether the change can be verified with a screenshot.
- If screenshot verification is possible, capture and inspect the screenshot first. Do not commit until the screenshot confirms the UI, notification, or other visual behavior is correct.
- Review the exact code and documentation diff that will be committed. Fix any issues found before committing.
- Push the feature branch and open a PR into `main`.
- After opening or updating the PR, review the PR diff and checks again. Apply any fixes in the same feature branch, push again, and repeat review.
- Review the PR. Apply requested changes in the same feature branch, push again, and repeat until review has no blocking issues.
- Keep the branch rebased on latest `main` before final merge.
- Prefer a linear main history. Use GitHub "Rebase and merge" when suitable, or rebase locally and fast-forward merge.
- Release is manual. Do not create or push version tags unless the user explicitly asks for a release.
- Not every merge to `main` needs a tag or release.
- After the PR is merged into `main`, the feature worktree can be removed.
- After removing the worktree, delete the local feature branch when it is no longer needed.
