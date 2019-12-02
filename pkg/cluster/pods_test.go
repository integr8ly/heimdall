package cluster_test

import (
	"context"
	"fmt"
	"github.com/integr8ly/heimdall/pkg/cluster"
	"github.com/integr8ly/heimdall/pkg/domain"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestPods_LabelPods(t *testing.T) {
	cases := []struct {
		Name      string
		Pod       *v1.Pod
		Report    *domain.ReportResult
		ExpectErr bool
		Validate  func(t *testing.T, p *v1.Pod)
	}{
		{
			Name: "test pod is labelled correctly when single container",
			Pod: &v1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test",
				},
				Status: v1.PodStatus{Phase: v1.PodRunning},
			},
			Report: &domain.ReportResult{
				ResolvableCVEs: []domain.CVE{domain.CVE{
					Severity:   "important",
					ID:         "id",
					AdvisoryID: "id",
				}},
				ClusterImage: &domain.ClusterImage{
					Pods: []domain.PodAndContainerRef{
						{
							Name:      "test-pod",
							Namespace: "test",
							Containers: []string{
								"container",
							},
						},
					},
				},
			},
			Validate: func(t *testing.T, p *v1.Pod) {
				if p == nil {
					t.Fatal("expected a pod but got none")
				}
				var val string
				val = p.Labels[cluster.LabelAggregateResolvableCritCVE]
				if val != "false" {
					t.Fatal("expected no critical CVEs	 on pod labels")
				}
				val = p.Labels[cluster.LabelAggregateResolvableImportantCVE]
				if val != "true" {
					t.Fatal("expected some important CVEs on pod labels")
				}
				val = p.Labels[cluster.LabelAggregateResolvableModerateCVE]
				if val != "false" {
					t.Fatal("expected no moderate CVEs on pod labels")
				}
			},
		},
		{
			Name: "test pod is labelled correctly when single image multiple containers using image",
			Pod: &v1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Namespace: "test",
					Name:      "test-pod",
				},
				Status: v1.PodStatus{Phase: v1.PodRunning},
			},
			Report: &domain.ReportResult{
				ResolvableCVEs: []domain.CVE{{
					Severity: "important",
				}, {
					Severity: "critical",
				}},
				ClusterImage: &domain.ClusterImage{
					Pods: []domain.PodAndContainerRef{
						{
							Name:      "test-pod",
							Namespace: "test",
							Containers: []string{
								"container1",
								"container2",
							},
						},
					},
				},
			},
			Validate: func(t *testing.T, p *v1.Pod) {
				if p == nil {
					t.Fatal("expected a pod but got none")
				}
				var val string
				val = p.Labels[fmt.Sprintf(cluster.LabelContainerFormat, "container1", "resolvableImportantCVEs")]
				if val != "1" {
					t.Fatal("expected 1 important cve but got ", val)
				}
				val = p.Labels[fmt.Sprintf(cluster.LabelContainerFormat, "container2", "resolvableImportantCVEs")]
				if val != "1" {
					t.Fatal("expected 1 important cve but got ", val)
				}
				val = p.Labels[fmt.Sprintf(cluster.LabelContainerFormat, "container2", "resolvableCriticalCVEs")]
				if val != "1" {
					t.Fatal("expected 1 important cve but got ", val)
				}
				val = p.Labels[fmt.Sprintf(cluster.LabelContainerFormat, "container1", "resolvableCriticalCVEs")]
				if val != "1" {
					t.Fatal("expected 1 important cve but got ", val)
				}
				val = p.Labels[cluster.LabelAggregateResolvableCritCVE]
				if val != "true" {
					t.Fatal("expected critical CVEs	 on pod labels")
				}
				val = p.Labels[cluster.LabelAggregateResolvableImportantCVE]
				if val != "true" {
					t.Fatal("expected important CVEs on pod labels")
				}
				val = p.Labels[cluster.LabelAggregateResolvableModerateCVE]
				if val != "false" {
					t.Fatal("did not expect moderate CVEs on pod labels")
				}
			},
		},
		{
			Name: "test we get an error when we fail to find a pod",
			Pod: &v1.Pod{
				ObjectMeta: v12.ObjectMeta{},
			},
			Report: &domain.ReportResult{
				ClusterImage: &domain.ClusterImage{
					Pods: []domain.PodAndContainerRef{
						{
							Name:      "test-pod",
							Namespace: "test",
							Containers: []string{
								"container",
							},
						},
					},
				},
			},
			ExpectErr: true,
		},
		{
			Name: "test we get an error when pod not running",
			Pod: &v1.Pod{
				ObjectMeta: v12.ObjectMeta{},
				Status:     v1.PodStatus{Phase: v1.PodPending},
			},
			Report: &domain.ReportResult{
				ClusterImage: &domain.ClusterImage{
					Pods: []domain.PodAndContainerRef{
						{
							Name:      "test-pod",
							Namespace: "test",
							Containers: []string{
								"container",
							},
						},
					},
				},
			},
			ExpectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			client := fakeclient.NewFakeClient(tc.Pod)
			pods := cluster.NewPods(client)
			err := pods.LabelPods(tc.Report)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected and error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if err := client.Get(context.TODO(), client2.ObjectKey{Namespace: tc.Pod.Namespace, Name: tc.Pod.Name}, tc.Pod); err != nil {
				t.Fatal(err, "expected to get a pod but got an error ")
			}
			if tc.Validate != nil {
				tc.Validate(t, tc.Pod)
			}
		})
	}
}
