package commands_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"
)

func parseTime(timeString string) time.Time {
	timeValue, err := time.Parse(time.RFC3339, timeString)

	if err != nil {
		return time.Time{}
	}

	return timeValue
}

var _ = FDescribe("Installations", func() {
	var (
		command                  commands.Installations
		fakeInstallationsService *fakes.InstallationsService
		tableWriter              *fakes.TableWriter
	)

	BeforeEach(func() {
		tableWriter = &fakes.TableWriter{}
		fakeInstallationsService = &fakes.InstallationsService{}
		command = commands.NewInstallations(fakeInstallationsService, tableWriter)
	})

	Describe("Execute", func() {
		It("lists recent installations as a table", func() {
			fakeInstallationsService.ListInstallationsReturns([]api.InstallationsServiceOutput{
				{
					ID:         1,
					UserName:   "some-user",
					Status:     "succeeded",
					StartedAt:  parseTime("2017-05-24T23:38:37.316Z"),
					FinishedAt: parseTime("2017-05-24T23:39:37.316Z"),
				},
				{
					ID:         2,
					UserName:   "some-user2",
					Status:     "failed",
					StartedAt:  parseTime("2017-05-25T23:38:37.316Z"),
					FinishedAt: parseTime("2017-05-25T23:39:37.316Z"),
				},
			}, nil)

			err := command.Execute()
			Expect(err).NotTo(HaveOccurred())

			Expect(tableWriter.SetHeaderCallCount()).To(Equal(1))
			Expect(tableWriter.SetHeaderArgsForCall(0)).To(Equal([]string{"ID", "User", "Status", "Started At", "Finished At"}))

			Expect(tableWriter.AppendCallCount()).To(Equal(2))
			Expect(tableWriter.AppendArgsForCall(0)).To(Equal([]string{"1", "some-user", "succeeded", "2017-05-24T23:38:37.316Z", "2017-05-24T23:39:37.316Z"}))
			Expect(tableWriter.AppendArgsForCall(1)).To(Equal([]string{"2", "some-user2", "failed", "2017-05-25T23:38:37.316Z", "2017-05-25T23:39:37.316Z"}))

			Expect(tableWriter.RenderCallCount()).To(Equal(1))
		})
	})
})
