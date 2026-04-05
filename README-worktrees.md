# TLDR
1. Create the work tree ```git worktree ai-agent-task -b branchName```
2. Commit work from agent ```git commit (from ai agent)```
3. merge work from one to the other ```git merge worktreename```
4. list current worktrees ```git worktree list```
# Worktrees for AI Agents and Collaborators

Purpose
- This document describes how to use Git worktrees to have AI agents and human users collaborate on the same feature without stepping on each other's changes. Each worktree provides an isolated checkout of a feature branch so changes can be developed in parallel and merged via pull requests.

Prerequisites
- Git (version 2.5+ recommended)
- A clone of this repository with a remote (origin)
- Basic familiarity with Git worktrees

Naming conventions
- Feature branches: feature/<name>
- Worktree directories: worktrees/<name>
- Example feature name: feature/improve-vector-search

Getting started
- Ensure your main branch is up to date:

  git fetch origin
  git checkout main
  git pull --ff-only

- Create a new worktree for a feature (new or existing branch):

  # If this is a new feature branch based on main
  git worktree add -b feature/improve-vector-search worktrees/feature-improve-vector-search main

  # If the feature branch already exists remotely, you can base the worktree on it
  # git fetch origin
  # git worktree add -b feature/improve-vector-search worktrees/feature-improve-vector-search origin/feature/improve-vector-search

- Work in the new directory:

  cd worktrees/feature-improve-vector-search
  # make your changes, then commit
  git add .
  git commit -m "feat: describe what you changed"
  git push -u origin feature/improve-vector-search

Syncing and coordination
- To keep the main branch updated, pull in the root repo:

  # In the root of the repository
  git pull --ff-only

- In each worktree, you can fetch or rebase onto updated main, or merge as appropriate:

  # From the worktree directory
  git fetch origin
  git rebase origin/main   # or: git merge origin/main

- When the work is ready for review, open a PR from feature/improve-vector-search.
- Cleaning up a finished worktree:

  # From the main repo
  git worktree remove worktrees/feature-improve-vector-search
  # Optionally remove the directory if empty

Best practices
- Keep one feature per worktree; this helps avoid merge conflicts and makes PRs clearer.
- Coordinate with other collaborators or agents to avoid editing the same files in parallel.
- Always run tests locally in the worktree before pushing.
- Name worktrees clearly and consistently to reflect the feature and (if needed) the contributor.

Troubleshooting
- If a worktree becomes out of date or inconsistent, re-run:
  git fetch origin
  git rebase origin/main   # or merge
- If a worktree path is missing, you can recreate it using the same commands above.

Notes
- This repository uses a single main branch as the source of truth. Feature work is isolated in dedicated worktrees and merged via PRs.
- AI agents can run in their own worktrees to apply changes to a feature independently from human collaborators.

End of guide
