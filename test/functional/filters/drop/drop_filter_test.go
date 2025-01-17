package drop

import (
	"time"

	"github.com/openshift/cluster-logging-operator/test/framework/functional"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
)

var _ = Describe("[Functional][Filters][Drop] Drop filter", func() {
	const (
		dropFilterName = "myDrop"
	)

	var (
		f *functional.CollectorFunctionalFramework
	)

	AfterEach(func() {
		f.Cleanup()
	})

	Describe("when drop filter is spec'd", func() {
		It("should drop logs that have `error` in its message OR logs with messages that doesn't include `information` AND includes `debug`", func() {
			f = functional.NewCollectorFunctionalFrameworkUsingCollector(logging.LogCollectionTypeVector)
			f.Forwarder.Spec.Filters = []logging.FilterSpec{
				{
					Name: dropFilterName,
					Type: logging.FilterDrop,
					FilterTypeSpec: logging.FilterTypeSpec{
						DropTestsSpec: &[]logging.DropTest{
							{
								DropConditions: []logging.DropCondition{
									{
										Field:   ".message",
										Matches: "error",
									},
								},
							},
							{
								DropConditions: []logging.DropCondition{
									{
										Field:      ".message",
										NotMatches: "information",
									},
									{
										Field:   ".message",
										Matches: "debug",
									},
								},
							},
						},
					},
				},
			}
			functional.NewClusterLogForwarderBuilder(f.Forwarder).
				FromInput(logging.InputNameApplication).
				ToElasticSearchOutput()

			f.Forwarder.Spec.Pipelines = []logging.PipelineSpec{
				{
					Name:       "myDropPipeline",
					FilterRefs: []string{dropFilterName},
					InputRefs:  []string{logging.InputNameApplication, logging.InputNameAudit, logging.InputNameInfrastructure},
					OutputRefs: []string{logging.OutputTypeElasticsearch},
				},
			}

			Expect(f.Deploy()).To(BeNil())
			msg := functional.NewFullCRIOLogMessage(functional.CRIOTime(time.Now()), "my error message")
			Expect(f.WriteMessagesToApplicationLog(msg, 1)).To(BeNil())
			msg2 := functional.NewFullCRIOLogMessage(functional.CRIOTime(time.Now()), "information message")
			Expect(f.WriteMessagesToApplicationLog(msg2, 1)).To(BeNil())
			msg3 := functional.NewFullCRIOLogMessage(functional.CRIOTime(time.Now()), "debug message")
			Expect(f.WriteMessagesToApplicationLog(msg3, 1)).To(BeNil())
			Expect(f.WritesApplicationLogs(5)).To(Succeed())

			logs, err := f.ReadApplicationLogsFrom(logging.OutputTypeElasticsearch)
			Expect(err).To(BeNil(), "Error fetching logs from %s: %v", logging.OutputTypeElasticsearch, err)
			Expect(logs).To(Not(BeEmpty()), "Exp. logs to be forwarded to %s", logging.OutputTypeElasticsearch)
			hasInfoMessage := false
			for _, msg := range logs {
				Expect(msg.ViaQCommon.Message).ToNot(Equal("my error message"))
				Expect(msg.ViaQCommon.Message).ToNot(Equal("debug message"))
				if msg.ViaQCommon.Message == "information message" {
					hasInfoMessage = true
				}
			}
			Expect(hasInfoMessage).To(BeTrue())
		})

	})
})
