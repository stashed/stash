---
name: Release Checklist
about: Check list for a release process
title: 'Release Checklist: [X.X.X]'
labels: ''
assignees: ''

---

**Step-1: Stash Operator**
- [ ] Release Stash Operator
- [ ] Publish Stash Docker Image
- [ ] Update [stashed/installer](https://github.com/stashed/installer)
- [ ] Release [stashed/installer](https://github.com/stashed/installer)
- [ ] Publish Stash Chart

**Step-2: Stash Addons**
This step is required only if any change has been introduces since the last release or changes in Stash operator affect the addons.

- [ ] Revendor addons repos with latest Stash release
- [ ] Cherry-pick changes into all versions & re-tag
- [ ] Update [stashed/catalog](https://github.com/stashed/catalog) (if required)
- [ ] Release [stashed/catalog](https://github.com/stashed/catalog) (if required)
- [ ] Publish Addon Charts
- [ ] Publish Addon Docker Images

**Step-3: Documentation**
- [ ] Write Changelog in [stashed/docs](https://github.com/stashed/docs)
- [ ] Release [stashed/docs](https://github.com/stashed/docs)

**Step-4: Website Update**
- [ ] Update [appscode/static-assets](https://github.com/appscode/static-assets)
- [ ] Update [appscode-cloud/hugo-appscode](https://github.com/appscode-cloud/hugo-appscode)
- [ ] Render & Verify latest update locally
- [ ] Publish Website

**Step-5: Announcement**
- [ ] Write Release Notes
- [ ] Announce new release
