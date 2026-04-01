---
description: How to create a new release for SubSage
---

# Release Workflow

// turbo-all

## Prerequisites
- All changes committed and pushed to `main`
- `npm run build` passes in `frontend/`
- Changes locally verified (via `go run .` in `backend/`)

## Steps

### 1. Determine version number
- **Patch** (0.1.x): bug fixes, UX polish, i18n additions
- **Minor** (0.x.0): new features, significant UI changes
- **Major** (x.0.0): breaking changes, major redesigns

### 2. Update version strings in codebase
Replace all `vOLD` with `vNEW` in these files:
- `frontend/src/i18n/en.js` — `nav.version`
- `frontend/src/i18n/zh.js` — `nav.version`
- `frontend/src/components/Sidebar.svelte` — version text (2 places: desktop + drawer)
- `frontend/src/pages/Settings.svelte` — about section
- `frontend/src/pages/Login.svelte` — login page footer
- `docs/architecture.md` — current version line

Quick command:
```bash
cd frontend/src && sed -i '' 's/vOLD/vNEW/g' pages/Settings.svelte pages/Login.svelte i18n/zh.js i18n/en.js components/Sidebar.svelte
sed -i '' 's/vOLD/vNEW/' ../../docs/architecture.md
```

### 3. Add changelog entry to READMEs
Add a row to the version history table in both `README.md` and `README_zh.md`:
```
| vNEW | One-line summary of this release |
```

### 4. Rebuild frontend
```bash
cd frontend && npm run build
```

### 5. Commit and tag
```bash
git add -A
git commit -m "vNEW: one-line summary

Detailed changes..."
git tag vNEW
```

### 6. Push
```bash
git push origin main --tags
```

### 7. Create GitHub Release
Write release notes to a temp file, then:
```bash
gh release create vNEW --title "SubSage vNEW — One-Line Summary" --notes-file .release-notes.md
rm .release-notes.md
```

## Release Notes Template

```markdown
One sentence positioning this release.

## ✨ New Features
<!-- Include only when there are new features -->
- **Feature Name** — Description of what it does

## 🐛 Bug Fixes
<!-- Include only when there are bug fixes -->
- **Bug Name** — What was fixed and why (root cause if interesting)

## 📱 UX / UI
<!-- Include only when there are UI/UX improvements -->
- **Component** — What changed and why

## 🌐 i18n
<!-- Include only when there are i18n changes -->
- Added/Updated `key.name` (EN + ZH)

## ⚠️ Breaking Changes
<!-- Include only when there are breaking changes -->
- **What broke** — Migration steps

**Full Changelog**: https://github.com/wangjc683/subsage/compare/vPREV...vNEW
```

### Rules
- Title format: `SubSage vX.Y.Z — One-Line Summary`
- Opening line: one sentence describing the release focus
- Only include sections that have content
- Each item: `**Bold Name** — Description`
- End with Full Changelog compare link
