#!/bin/bash
# Update all references from beacon-runner-change-me to beacon-runner-production

set -e

OLD_URL="beacon-runner-change-me"
NEW_URL="beacon-runner-production"

echo "🔄 Updating Runner URL from $OLD_URL to $NEW_URL..."
echo ""

# Find and replace in all relevant files
find . -type f \( \
  -name "*.js" -o \
  -name "*.jsx" -o \
  -name "*.json" -o \
  -name "*.md" -o \
  -name "*.yml" -o \
  -name "*.yaml" \
\) -not -path "*/node_modules/*" -not -path "*/dist/*" -not -path "*/.git/*" \
  -exec grep -l "$OLD_URL" {} \; | while read file; do
    echo "📝 Updating: $file"
    sed -i '' "s/$OLD_URL/$NEW_URL/g" "$file"
done

echo ""
echo "✅ All files updated!"
echo ""
echo "📋 Summary of changes:"
git diff --stat

echo ""
echo "🔍 Review changes with: git diff"
echo "✅ Commit with: git add -A && git commit -m 'Update all Runner URLs to beacon-runner-production'"
