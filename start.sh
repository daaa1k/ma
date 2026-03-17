#!/usr/bin/env bash
# start.sh
#
# One-time setup script: replaces template placeholders with your project info.
# Run this once after creating a new repository from this template.
#
# Usage:
#   ./start.sh <app-name> <github-owner>
#
# Example:
#   ./start.sh mytools johndoe

set -Eeuo pipefail

trap 'echo "error: setup failed at line $LINENO" >&2' ERR

########################################
# args
########################################

APP_NAME="${1:?Usage: ./start.sh <app-name> <github-owner>}"
GITHUB_OWNER="${2:?Usage: ./start.sh <app-name> <github-owner>}"

PLACEHOLDER_APP="myapp"
PLACEHOLDER_OWNER="daaa1k"

########################################
# validation
########################################

if [[ ! "$APP_NAME" =~ ^[a-z][a-z0-9_-]*$ ]]; then
  echo "error: app-name must be lowercase alphanumeric (hyphens/underscores allowed)" >&2
  exit 1
fi

if [[ ! "$GITHUB_OWNER" =~ ^[a-zA-Z0-9_-]+$ ]]; then
  echo "error: github-owner must be alphanumeric" >&2
  exit 1
fi

########################################
# helpers
########################################

sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

# Capitalize first letter: myapp -> Myapp
capitalize() {
  echo "${1:0:1}" | tr '[:lower:]' '[:upper:]'
  echo "${1:1}"
}

APP_NAME_CAP="$(capitalize "$APP_NAME" | tr -d '\n')"
PLACEHOLDER_APP_CAP="Myapp"

########################################
# replace in files
########################################

echo "==> Replacing placeholders in files"

# Files that contain placeholders (excluding .git and this script itself)
FILES=$(grep -rl \
  --exclude="start.sh" \
  --exclude-dir=".git" \
  --exclude-dir=".agents" \
  -e "$PLACEHOLDER_APP" \
  -e "$PLACEHOLDER_OWNER" \
  . 2>/dev/null || true)

for f in $FILES; do
  sedi \
    -e "s|${PLACEHOLDER_APP_CAP}|${APP_NAME_CAP}|g" \
    -e "s|${PLACEHOLDER_APP}|${APP_NAME}|g" \
    -e "s|${PLACEHOLDER_OWNER}|${GITHUB_OWNER}|g" \
    "$f"
  echo "  updated: $f"
done

########################################
# rename HomebrewFormula file
########################################

FORMULA_OLD="HomebrewFormula/${PLACEHOLDER_APP}.rb"
FORMULA_NEW="HomebrewFormula/${APP_NAME}.rb"

if [[ -f "$FORMULA_OLD" ]]; then
  mv "$FORMULA_OLD" "$FORMULA_NEW"
  echo "  renamed: $FORMULA_OLD -> $FORMULA_NEW"
fi

########################################
# activate .github_template -> .github
########################################

if [[ -d ".github_template" ]]; then
  if [[ -d ".github" ]]; then
    echo "  warning: .github already exists — merging .github_template into it"
    cp -rn .github_template/. .github/
    rm -rf .github_template
  else
    mv .github_template .github
  fi
  echo "  activated: .github_template -> .github"
fi

########################################
# make scripts executable
########################################

chmod +x scripts/*.sh 2>/dev/null || true

########################################
# self-delete
########################################

echo "==> Cleaning up start.sh"
rm -- "$0"

########################################
# done
########################################

echo ""
echo "Setup complete!"
echo "  App:   ${APP_NAME}"
echo "  Owner: ${GITHUB_OWNER}"
echo "  Module: github.com/${GITHUB_OWNER}/${APP_NAME}"
echo ""
echo "Next steps:"
echo "  1. Update go.mod dependencies as needed"
echo "  2. Fill in HomebrewFormula/${APP_NAME}.rb description and sha256 on first release"
echo "  3. git add . && git commit -m 'chore: initialize from template'"
