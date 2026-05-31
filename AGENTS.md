@/Users/gaoguobin/.codex/RTK.md

@/Users/gaoguobin/.codex/skills/caveman/SKILL.md

Default response mode: caveman full.
Use caveman mode from session start unless user says "stop caveman" or "normal mode".

## Repository Workflow

- Use git worktrees for feature work. Keep the repository root worktree on `main` long-term.
- Create one feature branch per task in its own worktree.
- Do development and verification inside the feature worktree.
- Push the feature branch and open a PR into `main`.
- Review the PR. Apply requested changes in the same feature branch, push again, and repeat until review has no blocking issues.
- Keep the branch rebased on latest `main` before final merge.
- Prefer a linear main history. Use GitHub "Rebase and merge" when suitable, or rebase locally and fast-forward merge.
- Release is manual. Do not create or push version tags unless the user explicitly asks for a release.
- Not every merge to `main` needs a tag or release.
- After the PR is merged into `main`, the feature worktree can be removed.
- After removing the worktree, delete the local feature branch when it is no longer needed.
