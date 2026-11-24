# Publish checklist

1. Run the full Go and npm test/build commands from README. Resolve failures before publishing a release.
2. Inspect `git status --short`. Commit `go.sum` and `web/package-lock.json` but never secrets, SQLite files, node_modules or binaries.
3. Create an empty GitHub repo or use the existing repo; preserve Git history. Push main normally, without force.
4. On GitHub, confirm the CI workflow is green. Enable Pages using GitHub Actions and run the Pages workflow. Verify the published preview from a browser. Add the actual Pages link to the README and repository About section.
5. Only after successful CI and manual testing, tag and push:

```bash
git tag -a v1.0.0-beta.1 -m "VeilWall integrated simulation beta"
git push origin v1.0.0-beta.1
```

6. Verify the prerelease assets and SHA256SUMS. Do not label this as stable or production-ready. Keep existing operating-system firewalls enabled.
