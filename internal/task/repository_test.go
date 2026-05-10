package task_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
	"github.com/philiplambok/tudu/internal/common/util"
	"github.com/philiplambok/tudu/internal/task"
)

const fixtureUserID = int64(1)

func newTask(title string) task.Task {
	return task.NewTask(task.CreateTaskRecordDTO{
		UserID:      fixtureUserID,
		Title:       title,
		Description: "test description",
	})
}

var _ = Describe("Repository", func() {
	var repo task.Repository

	BeforeEach(func() {
		err := testinfra.RestoreDB(ctx, container, &db)
		Expect(err).ToNot(HaveOccurred())
		repo = task.NewRepository(db)
	})

	Describe("Create", func() {
		It("persists the task with pending status and records a created activity", func() {
			result, err := repo.Create(ctx, newTask("Buy groceries"))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).ToNot(BeZero())
			Expect(result.UserID).To(Equal(fixtureUserID))
			Expect(result.Title).To(Equal("Buy groceries"))
			Expect(result.Status).To(Equal(task.StatusPending))
			Expect(result.CreatedAt).ToNot(BeZero())

			activities, err := repo.ListActivities(ctx, fixtureUserID, result.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(1))
			Expect(activities[0].Action).To(Equal(task.ActivityActionCreated))
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			_, err := repo.Create(ctx, newTask("Task A"))
			Expect(err).ToNot(HaveOccurred())
			_, err = repo.Create(ctx, newTask("Task B"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns all tasks for the user ordered by created_at DESC", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 20},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))
		})

		It("filters by status", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				Status:        task.StatusCompleted,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 20},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(records).To(BeEmpty())
		})

		It("respects limit and offset", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 1},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(1))
		})
	})

	Describe("Get", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Find me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the task for the correct user and ID", func() {
			result, err := repo.Get(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(created.ID))
			Expect(result.Title).To(Equal("Find me"))
		})

		It("returns ErrNotFound for the wrong userID", func() {
			_, err := repo.Get(ctx, 9999, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			_, err := repo.Get(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Update", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Old title"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("updates fields and records an activity for each changed field", func() {
			newTitle := "New title"
			agg := task.TaskFromRecord(*created)
			updated := agg.ApplyUpdate(task.UpdateRequestDTO{Title: &newTitle})

			result, err := repo.Update(ctx, fixtureUserID, updated)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Title).To(Equal("New title"))

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(2))
			Expect(activities[0].Action).To(Equal(task.ActivityActionUpdated))
			Expect(activities[0].FieldName).ToNot(BeNil())
			Expect(*activities[0].FieldName).To(Equal("title"))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			agg := task.Task{ID: 9999, UserID: fixtureUserID, Title: "x"}
			_, err := repo.Update(ctx, fixtureUserID, agg)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Complete", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Finish me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("sets status to completed and records completed_at", func() {
			result, err := repo.Complete(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Status).To(Equal(task.StatusCompleted))
			Expect(result.CompletedAt).ToNot(BeNil())
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			_, err := repo.Complete(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Delete me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes the task so it can no longer be retrieved", func() {
			err := repo.Delete(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(ctx, fixtureUserID, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			err := repo.Delete(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("ListActivities", func() {
		It("returns activities ordered by created_at DESC", func() {
			created, err := repo.Create(ctx, newTask("Activity task"))
			Expect(err).ToNot(HaveOccurred())

			newTitle := "Updated title"
			agg := task.TaskFromRecord(*created)
			updated := agg.ApplyUpdate(task.UpdateRequestDTO{Title: &newTitle})
			_, err = repo.Update(ctx, fixtureUserID, updated)
			Expect(err).ToNot(HaveOccurred())

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(2))
			Expect(activities[0].Action).To(Equal(task.ActivityActionUpdated))
			Expect(activities[1].Action).To(Equal(task.ActivityActionCreated))
		})

		It("returns ErrNotFound when the task does not belong to the user", func() {
			created, err := repo.Create(ctx, newTask("Private task"))
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.ListActivities(ctx, 9999, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Update with DueDate", func() {
		It("records a due_date activity when due_date changes", func() {
			dueDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
			agg := task.NewTask(task.CreateTaskRecordDTO{
				UserID:  fixtureUserID,
				Title:   "Dated task",
				DueDate: &dueDate,
			})
			created, err := repo.Create(ctx, agg)
			Expect(err).ToNot(HaveOccurred())

			newDue := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
			updated := task.TaskFromRecord(*created)
			withUpdate := updated.ApplyUpdate(task.UpdateRequestDTO{DueDate: &newDue})
			result, err := repo.Update(ctx, fixtureUserID, withUpdate)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.DueDate).ToNot(BeNil())

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(2))
			Expect(*activities[0].FieldName).To(Equal("due_date"))
		})
	})
})
