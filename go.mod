module github.com/CurtisD4q/jobwatch

go 1.21

require (
	github.com/CurtisD4q/mint-kit v0.0.0-20261117185925-ca8aae673810
	github.com/CurtisD4q/mint-loom v0.0.0-20261120195125-ce688870cc14
	github.com/CurtisD4q/mint-tools v0.0.0-20261120195125-ce688870cc14
)

// Until mint-loom's jobwatch screens are merged, the app builds against
// the working copy beside it.
replace github.com/CurtisD4q/mint-loom => ../ml
