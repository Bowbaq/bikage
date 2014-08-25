package bikage_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBikage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bikage Suite")
}
