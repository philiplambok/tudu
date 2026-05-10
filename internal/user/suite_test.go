package user_test

import (
	"context"
	"testing"

	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
)

var (
	db        *gorm.DB
	container *postgrescontainer.PostgresContainer
	ctx       context.Context
)

func TestUserRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User Repository Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	var err error
	db, container, err = testinfra.SetupTestDB(ctx)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if container != nil {
		err := container.Terminate(ctx)
		Expect(err).ToNot(HaveOccurred())
	}
})
