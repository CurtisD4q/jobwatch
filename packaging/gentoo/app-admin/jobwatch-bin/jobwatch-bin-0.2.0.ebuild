# Copyright 2026
# Distributed under the terms of the GNU General Public License v2
# jobwatch from its release archive. Run `ebuild … manifest` after
# changing the version.

EAPI=8

inherit elisp-common

DESCRIPTION="Are my backup and sync jobs still working, and will they keep working?"
HOMEPAGE="https://github.com/CurtisD4q/jobwatch"
SRC_URI="
	amd64? ( https://github.com/CurtisD4q/jobwatch/releases/download/v${PV}/jobwatch-${PV}-linux-amd64.tar.gz )
	arm64? ( https://github.com/CurtisD4q/jobwatch/releases/download/v${PV}/jobwatch-${PV}-linux-arm64.tar.gz )
"

LICENSE="Apache-2.0"
SLOT="0"
KEYWORDS="~amd64 ~arm64"
IUSE="emacs"
RESTRICT="strip"
QA_PREBUILT="usr/bin/jobwatch"

src_unpack() {
	default
	mv "${WORKDIR}"/jobwatch-${PV}-linux-* "${S}" || die
}

src_install() {
	dobin jobwatch
	dodoc README.md
	insinto /usr/share/licenses/${PN}
	doins -r LICENSE THIRD_PARTY
	if use emacs; then
		elisp-install ${PN} jobwatch.el
	fi
}
