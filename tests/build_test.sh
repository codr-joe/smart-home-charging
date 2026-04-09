#!/usr/bin/env bash
# Tests for Makefile container build targets.
# Each test function runs a specific assertion and reports PASS/FAIL.
# Exit code is non-zero when any test fails.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REGISTRY="harbor.hooyberghs.eu/smart-charging"
TAG="${TAG:-latest}"
API_IMAGE="${REGISTRY}/api:${TAG}"
WEB_IMAGE="${REGISTRY}/web:${TAG}"

PASS=0
FAIL=0

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

# ── Helpers ──────────────────────────────────────────────────────────────────

image_exists() {
  docker image inspect "$1" > /dev/null 2>&1
}

makefile_has_target() {
  make -C "$REPO_ROOT" -n "$1" > /dev/null 2>&1
}

# ── Target existence tests ────────────────────────────────────────────────────

test_makefile_targets_defined() {
  echo "Makefile targets defined:"
  for target in build build-api build-web push push-api push-web release; do
    if makefile_has_target "$target"; then
      pass "target '$target' is defined"
    else
      fail "target '$target' is missing from Makefile"
    fi
  done
}

# ── Dry-run tests (no Docker daemon required) ─────────────────────────────────

test_build_api_dry_run() {
  echo "build-api dry-run:"
  output=$(make -C "$REPO_ROOT" -n build-api 2>&1)
  if echo "$output" | grep -q "docker build"; then
    pass "build-api invokes docker build"
  else
    fail "build-api does not invoke docker build"
  fi
  if echo "$output" | grep -q "$API_IMAGE"; then
    pass "build-api uses correct API image tag"
  else
    fail "build-api does not reference image '$API_IMAGE'"
  fi
}

test_build_web_dry_run() {
  echo "build-web dry-run:"
  output=$(make -C "$REPO_ROOT" -n build-web 2>&1)
  if echo "$output" | grep -q "docker build"; then
    pass "build-web invokes docker build"
  else
    fail "build-web does not invoke docker build"
  fi
  if echo "$output" | grep -q "$WEB_IMAGE"; then
    pass "build-web uses correct web image tag"
  else
    fail "build-web does not reference image '$WEB_IMAGE'"
  fi
}

test_push_api_dry_run() {
  echo "push-api dry-run:"
  output=$(make -C "$REPO_ROOT" -n push-api 2>&1)
  if echo "$output" | grep -q "docker push"; then
    pass "push-api invokes docker push"
  else
    fail "push-api does not invoke docker push"
  fi
  if echo "$output" | grep -q "$API_IMAGE"; then
    pass "push-api uses correct API image tag"
  else
    fail "push-api does not reference image '$API_IMAGE'"
  fi
}

test_push_web_dry_run() {
  echo "push-web dry-run:"
  output=$(make -C "$REPO_ROOT" -n push-web 2>&1)
  if echo "$output" | grep -q "docker push"; then
    pass "push-web invokes docker push"
  else
    fail "push-web does not invoke docker push"
  fi
  if echo "$output" | grep -q "$WEB_IMAGE"; then
    pass "push-web uses correct web image tag"
  else
    fail "push-web does not reference image '$WEB_IMAGE'"
  fi
}

test_release_dry_run() {
  echo "release dry-run (build + push):"
  output=$(make -C "$REPO_ROOT" -n release 2>&1)
  if echo "$output" | grep -q "docker build"; then
    pass "release triggers docker build"
  else
    fail "release does not trigger docker build"
  fi
  if echo "$output" | grep -q "docker push"; then
    pass "release triggers docker push"
  else
    fail "release does not trigger docker push"
  fi
}

# ── Build tests (requires Docker daemon) ─────────────────────────────────────

test_build_api_image() {
  echo "build-api image:"
  # Remove any previous image to ensure a clean build
  docker rmi "$API_IMAGE" > /dev/null 2>&1 || true

  if make -C "$REPO_ROOT" build-api > /dev/null 2>&1; then
    pass "build-api completed successfully"
  else
    fail "build-api failed"
    return
  fi

  if image_exists "$API_IMAGE"; then
    pass "API image '$API_IMAGE' exists after build"
  else
    fail "API image '$API_IMAGE' not found after build"
  fi
}

test_build_web_image() {
  echo "build-web image:"
  docker rmi "$WEB_IMAGE" > /dev/null 2>&1 || true

  if make -C "$REPO_ROOT" build-web > /dev/null 2>&1; then
    pass "build-web completed successfully"
  else
    fail "build-web failed"
    return
  fi

  if image_exists "$WEB_IMAGE"; then
    pass "web image '$WEB_IMAGE' exists after build"
  else
    fail "web image '$WEB_IMAGE' not found after build"
  fi
}

test_build_all_images() {
  echo "build (all) images:"
  docker rmi "$API_IMAGE" "$WEB_IMAGE" > /dev/null 2>&1 || true

  if make -C "$REPO_ROOT" build > /dev/null 2>&1; then
    pass "build completed successfully"
  else
    fail "build failed"
    return
  fi

  if image_exists "$API_IMAGE" && image_exists "$WEB_IMAGE"; then
    pass "all images exist after build"
  else
    fail "one or more images missing after build"
  fi
}

# ── Runner ────────────────────────────────────────────────────────────────────

echo "=== Makefile container build tests ==="
echo ""

echo "--- Target definition tests ---"
test_makefile_targets_defined
echo ""

echo "--- Dry-run tests ---"
test_build_api_dry_run
test_build_web_dry_run
test_push_api_dry_run
test_push_web_dry_run
test_release_dry_run
echo ""

echo "--- Docker build tests ---"
test_build_api_image
test_build_web_image
test_build_all_images
echo ""

echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

[ "$FAIL" -eq 0 ]
