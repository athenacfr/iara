#!/usr/bin/env bash
# Tests for cw launcher

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CW="$SCRIPT_DIR/../bin/cw"
TEST_DIR=$(mktemp -d)
PASS=0
FAIL=0

# ── Helpers ─────────────────────────────────────────────────────────────────
cleanup() {
  rm -rf "$TEST_DIR"
}
trap cleanup EXIT

setup_test_repos() {
  for name in myapp-api myapp-web billing-api data-service; do
    mkdir -p "$TEST_DIR/repos/$name"
    git -C "$TEST_DIR/repos/$name" init -q
    git -C "$TEST_DIR/repos/$name" commit --allow-empty -m "init" -q
  done
  # Non-git directory
  mkdir -p "$TEST_DIR/repos/plain-folder"
}

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "  PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $label"
    echo "    expected: '$expected'"
    echo "    actual:   '$actual'"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local label="$1" needle="$2" haystack="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "  PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $label"
    echo "    expected to contain: '$needle'"
    echo "    actual: '$haystack'"
    FAIL=$((FAIL + 1))
  fi
}

assert_not_empty() {
  local label="$1" value="$2"
  if [ -n "$value" ]; then
    echo "  PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $label (was empty)"
    FAIL=$((FAIL + 1))
  fi
}

# ── Source cw functions (without running main) ──────────────────────────────
eval "$(sed -e 's/^main "\$@"//' -e 's/^set -eo pipefail//' "$CW")"

# ── Setup ───────────────────────────────────────────────────────────────────
setup_test_repos
SCAN_DIRS=("$TEST_DIR/repos")
SESSION_DIR="$TEST_DIR/sessions"
PROJECTS_DIR="$TEST_DIR/projects"
mkdir -p "$SESSION_DIR" "$PROJECTS_DIR"

# ── Tests ───────────────────────────────────────────────────────────────────
echo "== discover_repos =="

repos_output=$(discover_repos)
assert_contains "finds myapp-api" "myapp-api" "$repos_output"
assert_contains "finds myapp-web" "myapp-web" "$repos_output"
assert_contains "finds billing-api" "billing-api" "$repos_output"
assert_contains "finds data-service" "data-service" "$repos_output"
assert_contains "finds plain-folder" "plain-folder" "$repos_output"

echo ""
echo "== is_git_repo =="

if is_git_repo "$TEST_DIR/repos/myapp-api"; then
  echo "  PASS: myapp-api is a git repo"
  PASS=$((PASS + 1))
else
  echo "  FAIL: myapp-api should be a git repo"
  FAIL=$((FAIL + 1))
fi

if ! is_git_repo "$TEST_DIR/repos/plain-folder"; then
  echo "  PASS: plain-folder is not a git repo"
  PASS=$((PASS + 1))
else
  echo "  FAIL: plain-folder should not be a git repo"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "== repo_branch =="

branch=$(repo_branch "$TEST_DIR/repos/myapp-api")
assert_not_empty "git repo returns a branch" "$branch"

branch=$(repo_branch "$TEST_DIR/repos/plain-folder")
assert_eq "non-git returns 'no git'" "no git" "$branch"

echo ""
echo "== format_repo_line =="

line=$(format_repo_line "$TEST_DIR/repos/myapp-api")
assert_contains "includes repo name" "myapp-api" "$line"
assert_contains "includes branch in parens" "(" "$line"

echo ""
echo "== save_project / load_project =="

save_project "testproj" "$TEST_DIR/repos/myapp-api/" "$TEST_DIR/repos/myapp-web/"
assert_eq "project file created" "true" "$([ -f "$PROJECTS_DIR/testproj.conf" ] && echo true || echo false)"

load_project "$PROJECTS_DIR/testproj.conf"
assert_eq "name loaded" "testproj" "$PROJECT_NAME"
assert_eq "primary loaded" "$TEST_DIR/repos/myapp-api/" "$PROJECT_PRIMARY"
assert_eq "add_dir loaded" "$TEST_DIR/repos/myapp-web/" "${PROJECT_ADD_DIRS[0]}"

echo ""
echo "== list_projects =="

save_project "proj-a" "$TEST_DIR/repos/myapp-api/"
save_project "proj-b" "$TEST_DIR/repos/billing-api/"
projects_list=$(list_projects)
assert_contains "lists proj-a" "proj-a" "$projects_list"
assert_contains "lists proj-b" "proj-b" "$projects_list"

echo ""
echo "== last_session_for =="

result=$(last_session_for "myapp-api")
assert_eq "no session returns empty" "" "$result"

touch "$SESSION_DIR/2026-03-11-myapp-api-session.tmp"
result=$(last_session_for "myapp-api")
assert_not_empty "finds session file" "$result"
assert_contains "shows time ago" "ago" "$result"

echo ""
echo "== worktree operations =="

# Create a worktree
wt_path=$(create_worktree "$TEST_DIR/repos/myapp-api" "feat-test" 2>/dev/null)
assert_not_empty "create_worktree returns path" "$wt_path"

if [ -f "$wt_path/.git" ]; then
  echo "  PASS: worktree has .git file (linked)"
  PASS=$((PASS + 1))
else
  echo "  FAIL: worktree should have .git file"
  FAIL=$((FAIL + 1))
fi

# List worktrees
wt_list=$(list_worktrees "$TEST_DIR/repos/myapp-api")
assert_contains "lists feat-test worktree" "feat-test" "$wt_list"

# Creating same worktree again returns existing path
wt_path2=$(create_worktree "$TEST_DIR/repos/myapp-api" "feat-test" 2>/dev/null)
assert_eq "idempotent worktree creation" "$wt_path" "$wt_path2"

# Cleanup
git -C "$TEST_DIR/repos/myapp-api" worktree remove "$wt_path" 2>/dev/null

echo ""
echo "== MODE_FILES =="

assert_not_empty "research mode has file" "${MODE_FILES[research]:-}"
assert_not_empty "debug mode has file" "${MODE_FILES[debug]:-}"
assert_not_empty "review mode has file" "${MODE_FILES[review]:-}"

for m in research debug review; do
  f="${MODE_FILES[$m]}"
  if [ -f "$f" ]; then
    echo "  PASS: $m mode file exists"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $m mode file missing at $f"
    FAIL=$((FAIL + 1))
  fi
done

echo ""
echo "== MODE_COMMANDS =="

assert_eq "plan mode triggers /plan" "/plan" "${MODE_COMMANDS[plan]}"
assert_eq "tdd mode triggers /tdd" "/tdd" "${MODE_COMMANDS[tdd]}"

echo ""
echo "== show_usage =="

usage_output=$(show_usage)
assert_contains "usage shows cw" "cw" "$usage_output"
assert_contains "usage shows modes" "MODES" "$usage_output"
assert_contains "usage shows project flow" "project" "$usage_output"

echo ""
echo "========================================="
echo "Results: $PASS passed, $FAIL failed"
echo "========================================="

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
