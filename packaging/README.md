# Packaging

Both packages install the release archive, so they need nothing private to build.

- **Arch (AUR):** `packaging/aur/PKGBUILD` for `jobwatch-bin`. After a release, set `pkgver` and run `updpkgsums`.
- **Gentoo:** `packaging/gentoo/app-admin/jobwatch-bin/`. Copy it into an overlay, rename the ebuild to the new version, and run `ebuild jobwatch-bin-VERSION.ebuild manifest`.
