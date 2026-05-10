package user_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
	"github.com/philiplambok/tudu/internal/user"
)

var _ = Describe("Repository", func() {
	var repo user.Repository

	BeforeEach(func() {
		err := testinfra.RestoreDB(ctx, container, &db)
		Expect(err).ToNot(HaveOccurred())
		repo = user.NewRepository(db)
	})

	Describe("Create", func() {
		It("creates a user and returns the DTO without exposing the password hash", func() {
			result, err := repo.Create(ctx, user.CreateUserRecordDTO{
				Email:        "new@example.com",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).ToNot(BeZero())
			Expect(result.Email).To(Equal("new@example.com"))
			Expect(result.CreatedAt).ToNot(BeZero())
		})

		It("returns ErrEmailConflict when the email already exists", func() {
			_, err := repo.Create(ctx, user.CreateUserRecordDTO{
				Email:        "testuser@example.com",
				PasswordHash: "hash",
			})
			Expect(err).To(Equal(user.ErrEmailConflict))
		})
	})

	Describe("FindByEmailForAuth", func() {
		It("returns the auth record for an existing email", func() {
			result, err := repo.FindByEmailForAuth(ctx, "testuser@example.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int64(1)))
			Expect(result.Email).To(Equal("testuser@example.com"))
			Expect(result.PasswordHash).ToNot(BeEmpty())
		})

		It("returns ErrNotFound for an unknown email", func() {
			_, err := repo.FindByEmailForAuth(ctx, "nobody@example.com")
			Expect(err).To(Equal(user.ErrNotFound))
		})
	})

	Describe("FindByID", func() {
		It("returns the user DTO for an existing ID", func() {
			result, err := repo.FindByID(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int64(1)))
			Expect(result.Email).To(Equal("testuser@example.com"))
		})

		It("returns ErrNotFound for an unknown ID", func() {
			_, err := repo.FindByID(ctx, 9999)
			Expect(err).To(Equal(user.ErrNotFound))
		})
	})
})
