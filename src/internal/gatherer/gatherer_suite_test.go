package gatherer_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gatherer Suite")
}
